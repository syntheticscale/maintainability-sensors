package sensors

import (
	"context"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// ParseTypeScriptTreeSitter parses a TypeScript file using tree-sitter and returns violations.
func ParseTypeScriptTreeSitter(filePath string) ([]Violation, error) {
	var violations []Violation

	content, err := os.ReadFile(filePath)
	if err != nil {
		return violations, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return violations, err
	}

	rootNode := tree.RootNode()

	// Walk the tree
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration", "method_definition", "arrow_function", "function_expression":
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
				for i := 0; i < int(parametersNode.NamedChildCount()); i++ {
					argCount++
				}
			} else if node.Type() == "arrow_function" {
				// Arrow functions can have a single identifier as parameter e.g. `x => x * 2`
				// We check if the first child is an identifier
				if node.ChildCount() > 0 {
					firstChild := node.Child(0)
					if firstChild.Type() == "identifier" {
						argCount = 1
					}
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

			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				var countComplexity func(n *sitter.Node)
				countComplexity = func(n *sitter.Node) {
					switch n.Type() {
					case "if_statement", "for_statement", "for_in_statement", "while_statement", "do_statement", "switch_case", "catch_clause", "ternary_expression":
						complexity++
					}
					for i := 0; i < int(n.NamedChildCount()); i++ {
						child := n.NamedChild(i)
						if child.Type() == "function_declaration" || child.Type() == "method_definition" || child.Type() == "arrow_function" || child.Type() == "function_expression" {
							continue // don't bleed into nested functions
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

		// Continue walking the rest of the tree
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}

	walk(rootNode)

	return violations, nil
}

// TypeScriptTreeSitterPlugin implements the Plugin interface for TypeScript using tree-sitter.
type TypeScriptTreeSitterPlugin struct{}

func (p TypeScriptTreeSitterPlugin) Name() string {
	return "typescript-ast"
}

func (p TypeScriptTreeSitterPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, filePath := range filePaths {
		violations, err := ParseTypeScriptTreeSitter(filePath)
		if err != nil {
			return nil, err
		}
		metricsMap[filePath] = violations
	}
	return metricsMap, nil
}
