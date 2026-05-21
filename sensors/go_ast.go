package sensors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// GoMetrics holds extracted metrics from a Go file.
type GoMetrics struct {
	Complexity     int
	FunctionLength int
	ArgumentCount  int
}

// ParseGoAST reads a Go file and extracts maintainability metrics natively.
func ParseGoAST(filePath string) ([]Violation, error) {
        var violations []Violation

        if info, err := os.Stat(filePath); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
                return violations, nil
        }

        fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return violations, err
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		startPos := fset.Position(fn.Body.Lbrace)
		endPos := fset.Position(fn.Body.Rbrace)
		startLine := startPos.Line
		endLine := endPos.Line

		// Calculate Function Length (lines of body)
		length := endLine - startLine + 1
		violations = append(violations, Violation{
			RuleName:  "FunctionLength",
			Value:     length,
			StartLine: startLine,
			EndLine:   endLine,
		})

		// Calculate Parameter (Argument) Count
		params := 0
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				if len(field.Names) > 0 {
					params += len(field.Names)
				} else {
					params++
				}
			}
		}
		violations = append(violations, Violation{
			RuleName:  "ArgumentCount",
			Value:     params,
			StartLine: startLine,
			EndLine:   endLine,
		})

		// Calculate Cyclomatic Complexity of the function
		complexity := calculateGoComplexity(fn)
		violations = append(violations, Violation{
			RuleName:  "Complexity",
			Value:     complexity,
			StartLine: startLine,
			EndLine:   endLine,
		})
	}

	return violations, nil
}

// calculateGoComplexity counts decision points in a Go function AST.
func calculateGoComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // base complexity is 1

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false // Do not leak complexity from nested closures
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			// Default clauses do not increase complexity
			if len(n.List) > 0 {
				complexity++
			}
		case *ast.CommClause:
			// Channel select cases
			if n.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}

// GoPlugin implements the Plugin interface for Go using native AST parsing.
type GoPlugin struct{}

func (p GoPlugin) Name() string {
	return "go-ast"
}

func (p GoPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, filePath := range filePaths {
		violations, err := ParseGoAST(filePath)
		if err != nil {
			return nil, err
		}
		metricsMap[filePath] = violations
	}
	return metricsMap, nil
}
