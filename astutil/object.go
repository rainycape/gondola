package astutil

import (
	"go/ast"
)

// ObjectName returns the object package and type
// name for the given ast.Object.
func ObjectName(obj *ast.Object) (pkg string, typ string) {
	switch decl := obj.Decl.(type) {
	case *ast.Field:
		return Selector(decl.Type)
	case *ast.AssignStmt:
		idx := -1
		for ii, v := range decl.Lhs {
			if Ident(v) == obj.Name {
				idx = ii
				break
			}
		}
		if idx >= 0 {
			return Selector(decl.Rhs[idx])
		}
	}
	return "", ""
}
