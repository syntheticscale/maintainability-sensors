package sensors

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// GoMetrics holds extracted metrics from a Go file.
type GoMetrics struct {
	Complexity     int
	FunctionLength int
	ArgumentCount  int
}

// ParseGoAST reads a Go file and extracts maintainability metrics natively.
func ParseGoAST(filePath string) (GoMetrics, error) {
	var metrics GoMetrics

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return metrics, err
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		// Calculate Function Length (lines of body)
		startPos := fset.Position(fn.Body.Lbrace)
		endPos := fset.Position(fn.Body.Rbrace)
		length := endPos.Line - startPos.Line
		if length > metrics.FunctionLength {
			metrics.FunctionLength = length
		}

		// Calculate Parameter (Argument) Count
		params := 0
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				// A field can define multiple identifiers of the same type: `a, b int`
				if len(field.Names) > 0 {
					params += len(field.Names)
				} else {
					params++ // anonymous parameter
				}
			}
		}
		if params > metrics.ArgumentCount {
			metrics.ArgumentCount = params
		}

		// Calculate Cyclomatic Complexity of the function
		complexity := calculateGoComplexity(fn)
		if complexity > metrics.Complexity {
			metrics.Complexity = complexity
		}
	}

	return metrics, nil
}

// calculateGoComplexity counts decision points in a Go function AST.
func calculateGoComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // base complexity is 1

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
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
			// Boolean logical AND / OR operators
			if n.Op == token.LAND || n.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}
