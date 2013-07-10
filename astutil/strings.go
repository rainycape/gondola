package astutil

import (
	"go/ast"
	"go/token"
	"gondola/pkg"
)

// Strings returns a list of string declarations of the given type
// (as a qualified name).
func Strings(fset *token.FileSet, f *ast.File, typ string) ([]ast.Expr, error) {
	pkg, tname := pkg.SplitQualifiedName(typ)
	pname, ok := Imports(f, pkg)
	if !ok {
		// Not imported
		return nil, nil
	}
	var strings []ast.Expr
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ValueSpec:
			p, t := Selector(x.Type)
			if p == pname && t == tname {
				for _, v := range x.Values {
					if s, pos := StringLiteral(fset, v); s != "" && pos != nil {
						strings = append(strings, v)
					}
				}
			}
		}
		return true
	})
	return strings, nil
}
