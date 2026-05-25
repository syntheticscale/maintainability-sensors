package sensors

import (
	"context"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ParsePythonTreeSitter parses a Python file using tree-sitter and returns violations.
func ParsePythonTreeSitter(file FileContext) ([]Violation, error) {
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
	filePath := file.Path

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return violations, err
	}

	rootNode := tree.RootNode()

	var imports []ImportInfo
	processPythonNode(rootNode, content, &imports, &violations)

	archConfig := findArchitectureConfig(filePath)
	if archConfig != nil {
		archViolations := CheckArchitectureDependencies(filePath, archConfig, imports)
		violations = append(violations, archViolations...)
	}

	return violations, nil
}

func processPythonNode(node *sitter.Node, content []byte, imports *[]ImportInfo, violations *[]Violation) {
	switch node.Type() {
	case "import_statement":
		extractPythonImport(node, content, imports)
	case "import_from_statement":
		extractPythonImportFrom(node, content, imports)
	case "function_definition":
		analyzePythonFunction(node, violations)
	}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		processPythonNode(node.NamedChild(i), content, imports, violations)
	}
}

func extractPythonImport(node *sitter.Node, content []byte, imports *[]ImportInfo) {
	startLine := int(node.StartPoint().Row) + 1
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "dotted_name" {
			addPythonImport(child, content, startLine, imports)
			continue
		}
		if child.Type() == "aliased_import" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				addPythonImport(nameNode, content, startLine, imports)
			}
		}
	}
}

func addPythonImport(node *sitter.Node, content []byte, startLine int, imports *[]ImportInfo) {
	sourceVal := string(content[node.StartByte():node.EndByte()])
	sourceVal = strings.ReplaceAll(sourceVal, ".", "/")
	*imports = append(*imports, ImportInfo{Path: sourceVal, Line: startLine})
}

func extractPythonImportFrom(node *sitter.Node, content []byte, imports *[]ImportInfo) {
	startLine := int(node.StartPoint().Row) + 1
	moduleNameNode := node.ChildByFieldName("module_name")
	if moduleNameNode != nil {
		addPythonImport(moduleNameNode, content, startLine, imports)
	}
}

func getPythonFunctionLength(node *sitter.Node) int {
	startLine := int(node.StartPoint().Row) + 1
	endLine := int(node.EndPoint().Row) + 1
	length := endLine - startLine + 1

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil || int(bodyNode.NamedChildCount()) == 0 {
		return length
	}

	firstStmt := bodyNode.NamedChild(0)
	if firstStmt.Type() != "expression_statement" || int(firstStmt.NamedChildCount()) == 0 {
		return length
	}

	if firstStmt.NamedChild(0).Type() == "string" {
		docStart := int(firstStmt.StartPoint().Row) + 1
		docEnd := int(firstStmt.EndPoint().Row) + 1
		length -= docEnd - docStart + 1
	}
	return length
}

func analyzePythonFunction(node *sitter.Node, violations *[]Violation) {
	startLine := int(node.StartPoint().Row) + 1
	endLine := int(node.EndPoint().Row) + 1

	length := getPythonFunctionLength(node)

	*violations = append(*violations, Violation{
		RuleName:  RuleFunctionLength,
		Value:     length,
		StartLine: startLine,
		EndLine:   endLine,
	})

	// Argument Count
	argCount := 0
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode != nil {
		argCount = int(parametersNode.NamedChildCount())
	}
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
		countPythonComplexity(bodyNode, &complexity)
	}

	*violations = append(*violations, Violation{
		RuleName:  RuleComplexity,
		Value:     complexity,
		StartLine: startLine,
		EndLine:   endLine,
	})
}

func countPythonComplexity(node *sitter.Node, complexity *int) {
	switch node.Type() {
	case "if_statement", "elif_clause", "for_statement", "while_statement", "except_clause",
		"try_statement", "with_statement",
		"boolean_operator",
		"list_comprehension", "dictionary_comprehension", "set_comprehension",
		"conditional_expression":
		*complexity++
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "function_definition" || child.Type() == "class_definition" {
			continue // don't bleed into nested functions/classes
		}
		countPythonComplexity(child, complexity)
	}
}

// PythonTreeSitterPlugin implements the Plugin interface for Python using tree-sitter.
type PythonTreeSitterPlugin struct{}

func (p PythonTreeSitterPlugin) Name() string {
	return "python-ast"
}

func (p PythonTreeSitterPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		violations, err := ParsePythonTreeSitter(file)
		if err != nil {
			return nil, err
		}
		metricsMap[file.Path] = violations
	}
	return metricsMap, nil
}
