package sensors

import (
	"context"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ParsePythonTreeSitter parses a Python file using tree-sitter and returns violations.
func ParsePythonTreeSitter(filePath string) ([]Violation, error) {
	var violations []Violation

	content, err := os.ReadFile(filePath)
	if err != nil {
		return violations, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return violations, err
	}

	rootNode := tree.RootNode()

	// Walk the tree
	var imports []ImportInfo
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		switch node.Type() {
		case "import_statement":
			startLine := int(node.StartPoint().Row) + 1
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child.Type() == "dotted_name" {
					sourceVal := string(content[child.StartByte():child.EndByte()])
					sourceVal = strings.ReplaceAll(sourceVal, ".", "/")
					imports = append(imports, ImportInfo{Path: sourceVal, Line: startLine})
				} else if child.Type() == "aliased_import" {
					nameNode := child.ChildByFieldName("name")
					if nameNode != nil {
						sourceVal := string(content[nameNode.StartByte():nameNode.EndByte()])
						sourceVal = strings.ReplaceAll(sourceVal, ".", "/")
						imports = append(imports, ImportInfo{Path: sourceVal, Line: startLine})
					}
				}
			}
		case "import_from_statement":
			startLine := int(node.StartPoint().Row) + 1
			moduleNameNode := node.ChildByFieldName("module_name")
			if moduleNameNode != nil {
				sourceVal := string(content[moduleNameNode.StartByte():moduleNameNode.EndByte()])
				sourceVal = strings.ReplaceAll(sourceVal, ".", "/")
				imports = append(imports, ImportInfo{Path: sourceVal, Line: startLine})
			}
		case "function_definition":
			startLine := int(node.StartPoint().Row) + 1
			endLine := int(node.EndPoint().Row) + 1

			// Function Length
			length := endLine - startLine + 1
			violations = append(violations, Violation{
				RuleName:  "FunctionLength",
				Value:     length,
				StartLine: startLine,
				EndLine:   endLine,
			})

			// Argument Count
			argCount := 0
			parametersNode := node.ChildByFieldName("parameters")
			if parametersNode != nil {
				// Count arguments. A parameter list has named arguments, etc.
				// For Python, standard parameters can be typed 'identifier' or 'typed_parameter' or 'default_parameter' etc.
				// The parameters node contains '(' 'identifier' ',' 'identifier' ')'
				// We can count children of parametersNode that are not '(', ')', or ','.
				for i := 0; i < int(parametersNode.NamedChildCount()); i++ {
					argCount++
				}
			}
			violations = append(violations, Violation{
				RuleName:  "ArgumentCount",
				Value:     argCount,
				StartLine: startLine,
				EndLine:   endLine,
			})

			// Complexity
			complexity := 1 // base

			// We need a helper to walk just the function body for complexity
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				var countComplexity func(n *sitter.Node)
				countComplexity = func(n *sitter.Node) {
					switch n.Type() {
					case "if_statement", "elif_clause", "for_statement", "while_statement", "except_clause":
						complexity++
					case "boolean_operator":
						// we might want to check if it's 'and' or 'or'
						// n.ChildByFieldName("operator")?
						// actually, for standard complexity we count these if we want, but let's see.
						// Our Go test just adds `if`, `for`, `while`, `except` and `elif`.
						// Let's stick to the basic nodes.
					}
					for i := 0; i < int(n.NamedChildCount()); i++ {
						child := n.NamedChild(i)
						if child.Type() == "function_definition" || child.Type() == "class_definition" {
							continue // don't bleed into nested functions/classes
						}
						countComplexity(child)
					}
				}
				countComplexity(bodyNode)
			}

			violations = append(violations, Violation{
				RuleName:  "Complexity",
				Value:     complexity,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}

		// Continue walking the rest of the tree (e.g. methods inside classes)
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}

	walk(rootNode)

	archConfig := findArchitectureConfig(filePath)
	if archConfig != nil {
		archViolations := CheckArchitectureDependencies(filePath, archConfig, imports)
		violations = append(violations, archViolations...)
	}

	return violations, nil
}

// PythonTreeSitterPlugin implements the Plugin interface for Python using tree-sitter.
type PythonTreeSitterPlugin struct{}

func (p PythonTreeSitterPlugin) Name() string {
	return "python-ast"
}

func (p PythonTreeSitterPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, filePath := range filePaths {
		violations, err := ParsePythonTreeSitter(filePath)
		if err != nil {
			return nil, err
		}
		metricsMap[filePath] = violations
	}
	return metricsMap, nil
}
