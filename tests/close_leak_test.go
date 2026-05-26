package tests

//nolint // maintainability: highly cohesive test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoMissingCloseCalls(t *testing.T) {
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".cache" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		// Find assignments like: f, err := os.Open(...)
		// and check if f.Close() is called.
		ast.Inspect(node, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}

			// We will look for identifiers that might need closing.
			needsClose := make(map[string]token.Pos)
			closed := make(map[string]bool)

			ast.Inspect(fn.Body, func(nn ast.Node) bool {
				switch stmt := nn.(type) {
				case *ast.AssignStmt:
					// simple heuristic: if right hand side is a call expression returning something that might need close
					// e.g. os.Open, os.Create, net.Dial, http.Get
					for _, rhs := range stmt.Rhs {
						if call, ok := rhs.(*ast.CallExpr); ok {
							if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
								methodName := sel.Sel.Name
								// common functions returning closable resources
								if methodName == "Open" || methodName == "Create" || methodName == "Get" || methodName == "Post" || methodName == "Dial" || methodName == "TempFile" || methodName == "CreateTemp" || methodName == "OpenFile" {
									for _, lhs := range stmt.Lhs {
										if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" && id.Name != "err" {
											needsClose[id.Name] = id.Pos()
										}
									}
								}
							}
						}
					}
				case *ast.DeferStmt:
					if call, ok := stmt.Call.Fun.(*ast.SelectorExpr); ok {
						if call.Sel.Name == "Close" {
							if id, ok := call.X.(*ast.Ident); ok {
								closed[id.Name] = true
							}
						}
					}
				case *ast.ExprStmt:
					// Also check regular f.Close() calls
					if call, ok := stmt.X.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							if sel.Sel.Name == "Close" {
								if id, ok := sel.X.(*ast.Ident); ok {
									closed[id.Name] = true
								}
							}
						}
					}
				}
				return true
			})

			for id, pos := range needsClose {
				if !closed[id] {
					t.Errorf("%s:%d: potential resource leak: %s is instantiated but never explicitly closed", fset.Position(pos).Filename, fset.Position(pos).Line, id)
				}
			}

			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
