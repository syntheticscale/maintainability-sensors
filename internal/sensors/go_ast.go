package sensors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// ParseGoAST reads a Go file and extracts maintainability metrics natively.
func ParseGoAST(file FileContext) ([]Violation, error) {
	if shouldSkipGoFile(file) {
		return nil, nil
	}

	fset := token.NewFileSet()
	f, err := parseGoFile(file, fset)
	if err != nil {
		return nil, err
	}

	if hasGoNolintComment(f) {
		return nil, nil
	}

	var violations []Violation
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
			violations = append(violations, analyzeGoFunction(fn, fset)...)
		}
	}

	return violations, nil
}

func shouldSkipGoFile(file FileContext) bool {
	if file.Content == nil {
		if info, err := os.Stat(file.Path); err == nil && (!info.Mode().IsRegular() || info.Size() > MaxFileSize) {
			return true
		}
	}
	return false
}

func parseGoFile(file FileContext, fset *token.FileSet) (*ast.File, error) {
	var src interface{}
	if file.Content != nil {
		src = file.Content
	}
	return parser.ParseFile(fset, file.Path, src, parser.ParseComments)
}

func hasGoNolintComment(f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//nolint") && strings.Contains(c.Text, "maintainability") {
				return true
			}
		}
	}
	return false
}

func analyzeGoFunction(fn *ast.FuncDecl, fset *token.FileSet) []Violation {
	var violations []Violation
	startLine := fset.Position(fn.Pos()).Line
	endLine := fset.Position(fn.End()).Line

	violations = append(violations, Violation{
		RuleName:  RuleFunctionLength,
		Value:     calculateGoFunctionLength(fn, fset),
		StartLine: startLine,
		EndLine:   endLine,
	})

	violations = append(violations, Violation{
		RuleName:  RuleArgumentCount,
		Value:     calculateGoParameterCount(fn),
		StartLine: startLine,
		EndLine:   endLine,
	})

	violations = append(violations, Violation{
		RuleName:  RuleComplexity,
		Value:     calculateGoComplexity(fn),
		StartLine: startLine,
		EndLine:   endLine,
	})

	violations = append(violations, Violation{
		RuleName:  RuleCognitiveComplexity,
		Value:     calculateGoCognitiveComplexity(fn),
		StartLine: startLine,
		EndLine:   endLine,
	})

	violations = append(violations, checkGoCaseBlockLength(fn, fset)...)

	return violations
}

func calculateGoFunctionLength(fn *ast.FuncDecl, fset *token.FileSet) int {
	bodyStartPos := fset.Position(fn.Body.Lbrace)
	bodyEndPos := fset.Position(fn.Body.Rbrace)
	return bodyEndPos.Line - bodyStartPos.Line + 1
}

func calculateGoParameterCount(fn *ast.FuncDecl) int {
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
	return params
}

func checkGoCaseBlockLength(fn *ast.FuncDecl, fset *token.FileSet) []Violation {
	var violations []Violation
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
				RuleName:  RuleCaseBlockLength,
				Value:     length,
				StartLine: caseStartLine,
				EndLine:   caseEndLine,
			})
		}
		return true
	})
	return violations
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
		violations, err := analyzeSingleGoFile(file)
		if err != nil {
			return nil, err
		}
		metricsMap[file.Path] = violations
	}
	return metricsMap, nil
}

func analyzeSingleGoFile(file FileContext) ([]Violation, error) {
	violations, err := ParseGoAST(file)
	if err != nil {
		return nil, err
	}

	if archCfg := findArchitectureConfig(file.Path); archCfg != nil {
		if archViolations, err := CheckGoArchitecture(file.Path, archCfg); err == nil && len(archViolations) > 0 {
			violations = append(violations, archViolations...)
		}
	}

	return violations, nil
}
