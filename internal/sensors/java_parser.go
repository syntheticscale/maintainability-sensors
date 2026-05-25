package sensors

import (
	"context"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func ParseJava(file FileContext) ([]Violation, error) {
	var violations []Violation

	var content []byte
	var err error
	if file.Content != nil {
		content = file.Content
	} else {
		if info, err := os.Stat(file.Path); err == nil && (!info.Mode().IsRegular() || info.Size() > MaxFileSize) {
			return violations, nil
		}
		content, err = os.ReadFile(file.Path)
		if err != nil {
			return violations, err
		}
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	tree, _ := parser.ParseCtx(context.Background(), nil, content)
	if tree == nil {
		return violations, nil
	}

	walkJavaNodes(tree.RootNode(), content, &violations)

	return violations, nil
}

func walkJavaNodes(n *sitter.Node, content []byte, violations *[]Violation) {
	if n == nil {
		return
	}

	t := n.Type()
	if t == "method_declaration" || t == "constructor_declaration" {
		startLine := int(n.StartPoint().Row) + 1
		endLine := int(n.EndPoint().Row) + 1
		length := endLine - startLine + 1

		*violations = append(*violations, Violation{RuleName: RuleFunctionLength, Value: length, StartLine: startLine, EndLine: endLine})
		*violations = append(*violations, Violation{RuleName: RuleArgumentCount, Value: countJavaParams(n), StartLine: startLine, EndLine: endLine})
		*violations = append(*violations, Violation{RuleName: RuleComplexity, Value: countJavaComplexity(n, content), StartLine: startLine, EndLine: endLine})
	}

	for i := 0; i < int(n.NamedChildCount()); i++ {
		walkJavaNodes(n.NamedChild(i), content, violations)
	}
}

func countJavaParams(c *sitter.Node) int {
	if c == nil {
		return 0
	}
	params := 0
	if c.Type() == "formal_parameter" || c.Type() == "spread_parameter" {
		params++
	}
	for i := 0; i < int(c.NamedChildCount()); i++ {
		params += countJavaParams(c.NamedChild(i))
	}
	return params
}

func countJavaComplexity(n *sitter.Node, content []byte) int {
	return 1 + sumJavaComplexity(n, content)
}

func sumJavaComplexity(c *sitter.Node, content []byte) int {
	if c == nil {
		return 0
	}
	complexity := 0
	ct := c.Type()
	switch ct {
	case "if_statement", "for_statement", "enhanced_for_statement", "while_statement", "do_statement", "switch_expression", "switch_statement", "catch_clause", "ternary_expression":
		complexity++
	case "binary_expression":
		complexity += countJavaBinaryComplexity(c, content)
	}
	for i := 0; i < int(c.NamedChildCount()); i++ {
		complexity += sumJavaComplexity(c.NamedChild(i), content)
	}
	return complexity
}

func countJavaBinaryComplexity(c *sitter.Node, content []byte) int {
	complexity := 0
	for i := 0; i < int(c.ChildCount()); i++ {
		complexity += checkJavaBinaryOp(c.Child(i), content)
	}
	return complexity
}

func checkJavaBinaryOp(child *sitter.Node, content []byte) int {
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

// JavaPlugin implements the Plugin interface for Java.
type JavaPlugin struct{}

func (p JavaPlugin) Name() string {
	return "java-ast"
}

func (p JavaPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		violations, err := ParseJava(file)
		if err != nil {
			return nil, err
		}
		metricsMap[file.Path] = violations
	}
	return metricsMap, nil
}
