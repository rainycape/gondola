package astutil

import (
	"go/ast"
	"go/token"
	"gnd.la/pkg"
	"strings"
)

// Calls returns the calls to a given function or method
// specified as a qualified name (e.g. gnd.la/i18n.T or
// gnd.la/mux.Context.T).
func Calls(fset *token.FileSet, f *ast.File, fn string) ([]*ast.CallExpr, error) {
	pkg, fname := pkg.SplitQualifiedName(fn)
	pname, ok := Imports(f, pkg)
	if !ok {
		// Not imported
		return nil, nil
	}
	dot := strings.Index(fname, ".")
	if dot == -1 {
		// function call
		return funcCalls(fset, f, pname, fname)
	}
	// method call
	return methodCalls(fset, f, pname, fname[:dot], fname[dot+1:])
}

func funcCalls(fset *token.FileSet, f *ast.File, pname, fname string) ([]*ast.CallExpr, error) {
	var calls []*ast.CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if ok && Ident(sel.X) == pname && Ident(sel.Sel) == fname {
				calls = append(calls, x)
			}
		}
		return true
	})
	return calls, nil
}

func methodCalls(fset *token.FileSet, f *ast.File, pname, tname, mname string) ([]*ast.CallExpr, error) {
	var calls []*ast.CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok || Ident(sel.Sel) != mname {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Obj != nil && id.Obj.Kind == ast.Var {
				p, t := ObjectName(id.Obj)
				if p == pname && t == tname {
					calls = append(calls, x)
				}
			}
		}
		return true
	})
	return calls, nil
}
