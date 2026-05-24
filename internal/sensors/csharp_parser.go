package sensors

import (
	"context"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
)

func ParseCSharp(file FileContext) ([]Violation, error) {
	var violations []Violation

	var content []byte
	var err error
	if file.Content != nil {
		content = file.Content
	} else {
		if info, err := os.Stat(file.Path); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
			return violations, nil
		}
		content, err = os.ReadFile(file.Path)
		if err != nil {
			return violations, err
		}
	}

	parser := sitter.NewParser()
	parser.SetLanguage(csharp.GetLanguage())

	tree, _ := parser.ParseCtx(context.Background(), nil, content)
	if tree == nil {
		return violations, nil
	}

	walkCSharpNodes(tree.RootNode(), content, &violations)

	return violations, nil
}

func walkCSharpNodes(n *sitter.Node, content []byte, violations *[]Violation) {
	if n == nil {
		return
	}

	t := n.Type()
	if t == "method_declaration" || t == "local_function_statement" || t == "constructor_declaration" {
		startLine := int(n.StartPoint().Row) + 1
		endLine := int(n.EndPoint().Row) + 1
		length := endLine - startLine + 1

		*violations = append(*violations, Violation{RuleName: "FunctionLength", Value: length, StartLine: startLine, EndLine: endLine})
		*violations = append(*violations, Violation{RuleName: "ArgumentCount", Value: countCSharpParams(n), StartLine: startLine, EndLine: endLine})
		*violations = append(*violations, Violation{RuleName: "Complexity", Value: countCSharpComplexity(n, content), StartLine: startLine, EndLine: endLine})
	}

	for i := 0; i < int(n.NamedChildCount()); i++ {
		walkCSharpNodes(n.NamedChild(i), content, violations)
	}
}

func countCSharpParams(c *sitter.Node) int {
	if c == nil {
		return 0
	}
	params := 0
	if c.Type() == "parameter" || c.Type() == "parameter_array" {
		params++
	}
	for i := 0; i < int(c.NamedChildCount()); i++ {
		params += countCSharpParams(c.NamedChild(i))
	}
	return params
}

func countCSharpComplexity(n *sitter.Node, content []byte) int {
	return 1 + sumCSharpComplexity(n, content)
}

func sumCSharpComplexity(c *sitter.Node, content []byte) int {
	if c == nil {
		return 0
	}
	complexity := 0
	ct := c.Type()
	switch ct {
	case "if_statement", "for_statement", "foreach_statement", "while_statement", "do_statement", "switch_statement", "catch_clause", "conditional_expression":
		complexity++
	case "binary_expression":
		complexity += countCSharpBinaryComplexity(c, content)
	}
	for i := 0; i < int(c.NamedChildCount()); i++ {
		complexity += sumCSharpComplexity(c.NamedChild(i), content)
	}
	return complexity
}

func countCSharpBinaryComplexity(c *sitter.Node, content []byte) int {
	complexity := 0
	for i := 0; i < int(c.ChildCount()); i++ {
		complexity += checkCSharpBinaryOp(c.Child(i), content)
	}
	return complexity
}

func checkCSharpBinaryOp(child *sitter.Node, content []byte) int {
	if child == nil || child.IsNamed() {
		return 0
	}
	start := child.StartByte()
	end := child.EndByte()
	if start >= end || end > uint32(len(content)) {
		return 0
	}
	op := string(content[start:end])
	if op == "&&" || op == "||" {
		return 1
	}
	return 0
}

// CSharpPlugin implements the Plugin interface for C#.
type CSharpPlugin struct{}

func (p CSharpPlugin) Name() string {
	return "csharp-ast"
}

func (p CSharpPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		violations, err := ParseCSharp(file)
		if err != nil {
			return nil, err
		}
		metricsMap[file.Path] = violations
	}
	return metricsMap, nil
}
