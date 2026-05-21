package sensors

import (
	"context"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func ParseJava(filePath string) ([]Violation, error) {
        var violations []Violation

        if info, err := os.Stat(filePath); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
                return violations, nil
        }

        content, err := os.ReadFile(filePath)
	if err != nil {
		return violations, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	tree, _ := parser.ParseCtx(context.Background(), nil, content)
	if tree == nil {
		return violations, nil
	}

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		t := n.Type()
		if t == "method_declaration" || t == "constructor_declaration" {
			startLine := int(n.StartPoint().Row) + 1
			endLine := int(n.EndPoint().Row) + 1

			// Length
			length := int(n.EndPoint().Row-n.StartPoint().Row) + 1
			violations = append(violations, Violation{
				RuleName:  "FunctionLength",
				Value:     length,
				StartLine: startLine,
				EndLine:   endLine,
			})

			// Params
			params := 0
			var countParams func(c *sitter.Node)
			countParams = func(c *sitter.Node) {
				if c == nil {
					return
				}
				if c.Type() == "formal_parameter" || c.Type() == "spread_parameter" {
					params++
				}
				for i := 0; i < int(c.NamedChildCount()); i++ {
					countParams(c.NamedChild(i))
				}
			}
			countParams(n)
			violations = append(violations, Violation{
				RuleName:  "ArgumentCount",
				Value:     params,
				StartLine: startLine,
				EndLine:   endLine,
			})

			// Complexity
			complexity := 1
			var countComplexity func(c *sitter.Node)
			countComplexity = func(c *sitter.Node) {
				if c == nil {
					return
				}
				ct := c.Type()
				if ct == "if_statement" || ct == "for_statement" || ct == "enhanced_for_statement" || ct == "while_statement" || ct == "do_statement" || ct == "switch_expression" || ct == "switch_statement" || ct == "catch_clause" {
					complexity++
				}
				if ct == "binary_expression" {
					for i := 0; i < int(c.ChildCount()); i++ {
						child := c.Child(i)
						if !child.IsNamed() {
							start := child.StartByte()
							end := child.EndByte()
							if start < end && end <= uint32(len(content)) {
								op := string(content[start:end])
								if op == "&&" || op == "||" {
									complexity++
								}
							}
						}
					}
				}
				if ct == "ternary_expression" {
					complexity++
				}
				for i := 0; i < int(c.NamedChildCount()); i++ {
					countComplexity(c.NamedChild(i))
				}
			}
			countComplexity(n)
			violations = append(violations, Violation{
				RuleName:  "Complexity",
				Value:     complexity,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}

		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}

	walk(tree.RootNode())

	return violations, nil
}

// JavaPlugin implements the Plugin interface for Java.
type JavaPlugin struct{}

func (p JavaPlugin) Name() string {
	return "java-ast"
}

func (p JavaPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, filePath := range filePaths {
		violations, err := ParseJava(filePath)
		if err != nil {
			return nil, err
		}
		metricsMap[filePath] = violations
	}
	return metricsMap, nil
}
