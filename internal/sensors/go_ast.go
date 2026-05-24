package sensors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// GoMetrics holds extracted metrics from a Go file.
type GoMetrics struct {
	Complexity     int
	FunctionLength int
	ArgumentCount  int
}

// ParseGoAST reads a Go file and extracts maintainability metrics natively.
func ParseGoAST(file FileContext) ([]Violation, error) {
	var violations []Violation

	if file.Content == nil {
		if info, err := os.Stat(file.Path); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
			return violations, nil
		}
	}

	var src interface{}
	if file.Content != nil {
		src = file.Content
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file.Path, src, 0)
	if err != nil {
		return violations, err
	}

	// Support file-level //nolint suppression
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//nolint") && strings.Contains(c.Text, "maintainability") {
				return violations, nil
			}
		}
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		startPos := fset.Position(fn.Pos())
		endPos := fset.Position(fn.End())
		startLine := startPos.Line
		endLine := endPos.Line

		// Calculate Function Length (lines of body)
		bodyStartPos := fset.Position(fn.Body.Lbrace)
		bodyEndPos := fset.Position(fn.Body.Rbrace)
		length := bodyEndPos.Line - bodyStartPos.Line + 1
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

		// Calculate Cognitive Complexity of the function
		cognitiveComplexity := calculateGoCognitiveComplexity(fn)
		violations = append(violations, Violation{
			RuleName:  "CognitiveComplexity",
			Value:     cognitiveComplexity,
			StartLine: startLine,
			EndLine:   endLine,
		})

		// Calculate Max Case Length
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			var caseStartLine, caseEndLine int
			switch node := n.(type) {
			case *ast.CaseClause:
				caseStartLine = fset.Position(node.Pos()).Line
				caseEndLine = fset.Position(node.End()).Line
			case *ast.CommClause:
				caseStartLine = fset.Position(node.Pos()).Line
				caseEndLine = fset.Position(node.End()).Line
			default:
				return true
			}
			length := caseEndLine - caseStartLine + 1
			if length > BaselineCaseLength {
				violations = append(violations, Violation{
					RuleName:  "CaseBlockLength",
					Value:     length,
					StartLine: caseStartLine,
					EndLine:   caseEndLine,
				})
			}
			return true
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

type cogVisitor struct {
	complexity *int
	nesting    int
}

func (v *cogVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		*v.complexity += 1 + v.nesting
		return &cogVisitor{
			complexity: v.complexity,
			nesting:    v.nesting + 1,
		}
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			*v.complexity++
		}
	case *ast.FuncLit:
		return nil // Do not leak complexity from nested closures
	}
	return v
}

// calculateGoCognitiveComplexity calculates cognitive complexity based on nesting depth.
func calculateGoCognitiveComplexity(fn *ast.FuncDecl) int {
	complexity := 0
	v := &cogVisitor{
		complexity: &complexity,
		nesting:    0,
	}
	ast.Walk(v, fn.Body)
	return complexity
}

// GoPlugin implements the Plugin interface for Go using native AST parsing.
type GoPlugin struct{}

func (p GoPlugin) Name() string {
	return "go-ast"
}

func (p GoPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		filePath := file.Path
		violations, err := ParseGoAST(file)
		if err != nil {
			return nil, err
		}

		if archCfg := findArchitectureConfig(filePath); archCfg != nil {
			if archViolations, err := CheckGoArchitecture(filePath, archCfg); err == nil && len(archViolations) > 0 {
				violations = append(violations, archViolations...)
			}
		}

		metricsMap[file.Path] = violations
	}
	return metricsMap, nil
}
