package sensors

import (
	"context"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// ParseTypeScriptTreeSitter parses a TypeScript file using tree-sitter and returns violations.
func ParseTypeScriptTreeSitter(file FileContext) ([]Violation, error) {
	var violations []Violation

	var content []byte
	var err error
	if file.Content != nil {
		content = file.Content
	} else {
		content, err = os.ReadFile(file.Path)
		if err != nil {
			return violations, err
		}
	}

	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return violations, err
	}

	rootNode := tree.RootNode()

	var imports []ImportInfo
	processTSNode(rootNode, content, &imports, &violations)

	archConfig := findArchitectureConfig(file.Path)
	if archConfig != nil {
		archViolations := CheckArchitectureDependencies(file.Path, archConfig, imports)
		violations = append(violations, archViolations...)
	}

	return violations, nil
}

func processTSNode(node *sitter.Node, content []byte, imports *[]ImportInfo, violations *[]Violation) {
	switch node.Type() {
	case "import_statement":
		extractTSImport(node, content, imports)
	case "call_expression":
		extractTSRequire(node, content, imports)
	case "function_declaration", "method_definition", "arrow_function", "function_expression":
		analyzeTSFunction(node, violations)
	}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		processTSNode(node.NamedChild(i), content, imports, violations)
	}
}

func extractTSImport(node *sitter.Node, content []byte, imports *[]ImportInfo) {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode != nil {
		startLine := int(sourceNode.StartPoint().Row) + 1
		sourceVal := string(content[sourceNode.StartByte():sourceNode.EndByte()])
		if len(sourceVal) >= 2 {
			sourceVal = sourceVal[1 : len(sourceVal)-1]
		}
		*imports = append(*imports, ImportInfo{Path: sourceVal, Line: startLine})
	}
}

func extractTSRequire(node *sitter.Node, content []byte, imports *[]ImportInfo) {
	functionNode := node.ChildByFieldName("function")
	if functionNode == nil || string(content[functionNode.StartByte():functionNode.EndByte()]) != "require" {
		return
	}

	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil || argsNode.NamedChildCount() == 0 {
		return
	}

	arg := argsNode.NamedChild(0)
	if arg.Type() != "string" {
		return
	}

	startLine := int(arg.StartPoint().Row) + 1
	sourceVal := string(content[arg.StartByte():arg.EndByte()])
	if len(sourceVal) >= 2 {
		sourceVal = sourceVal[1 : len(sourceVal)-1]
	}
	*imports = append(*imports, ImportInfo{Path: sourceVal, Line: startLine})
}

func analyzeTSFunction(node *sitter.Node, violations *[]Violation) {
	startLine := int(node.StartPoint().Row) + 1
	endLine := int(node.EndPoint().Row) + 1

	// Function Length
	length := endLine - startLine + 1
	*violations = append(*violations, Violation{
		RuleName:  RuleFunctionLength,
		Value:     length,
		StartLine: startLine,
		EndLine:   endLine,
	})

	// Argument Count
	argCount := getTSArgCount(node)
	*violations = append(*violations, Violation{
		RuleName:  RuleArgumentCount,
		Value:     argCount,
		StartLine: startLine,
		EndLine:   endLine,
	})

	// Complexity
	complexity := 1 // base
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		countTSComplexity(bodyNode, &complexity)
	}

	*violations = append(*violations, Violation{
		RuleName:  RuleComplexity,
		Value:     complexity,
		StartLine: startLine,
		EndLine:   endLine,
	})
}

func getTSArgCount(node *sitter.Node) int {
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode != nil {
		return int(parametersNode.NamedChildCount())
	}
	
	if node.Type() == "arrow_function" && node.ChildCount() > 0 {
		if node.Child(0).Type() == "identifier" {
			return 1
		}
	}
	return 0
}

func countTSComplexity(node *sitter.Node, complexity *int) {
	switch node.Type() {
	case "if_statement", "for_statement", "for_in_statement", "while_statement", "do_statement", "switch_case", "catch_clause", "ternary_expression":
		*complexity++
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "function_declaration" || child.Type() == "method_definition" || child.Type() == "arrow_function" || child.Type() == "function_expression" {
			continue // don't bleed into nested functions
		}
		countTSComplexity(child, complexity)
	}
}

// TypeScriptTreeSitterPlugin implements the Plugin interface for TypeScript using tree-sitter.
type TypeScriptTreeSitterPlugin struct{}

func (p TypeScriptTreeSitterPlugin) Name() string {
	return "typescript-ast"
}

func (p TypeScriptTreeSitterPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		violations, err := ParseTypeScriptTreeSitter(file)
		if err != nil {
			return nil, err
		}
		metricsMap[file.Path] = violations
	}
	return metricsMap, nil
}
