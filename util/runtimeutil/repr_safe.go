// +build appengine

package runtimeutil

// This is a repr implementation without using unsafe. It
// relies on the source code being present, otherwise it
// doesn't work.

import (
	"debug/gosym"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

func pointerRepr(val uint64, _ *gosym.Sym, _ bool) string {
	if val == 0 {
		return "= nil"
	}
	return "@ 0x" + strconv.FormatUint(val, 16)
}

func stringRepr(val1 uint64, val2 uint64) string {
	r := pointerRepr(val1, nil, false)
	if val1 != 0 {
		r += fmt.Sprintf(" (length = %d)", val2)
	}
	return r
}

func emptyInterfaceRepr(val1 uint64, val2 uint64) string {
	return pointerRepr(val2, nil, false)
}

func astTypeName(typ ast.Expr, pkg string) string {
	switch x := typ.(type) {
	case *ast.StarExpr:
		return "*" + astTypeName(x.X, pkg)
	case *ast.Ident:
		if basicType(x.Name) || pkg == "" {
			return x.Name
		}
		return pkg + "." + x.Name
	case *ast.SelectorExpr:
		return x.X.(*ast.Ident).Name + "." + x.Sel.Name
	case *ast.InterfaceType:
		return "interface {}"
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", astTypeName(x.Key, pkg), astTypeName(x.Value, pkg))
	case *ast.ArrayType:
		return fmt.Sprintf("[]%s", astTypeName(x.Elt, pkg))
	default:
		panic(fmt.Errorf("unhandled ast.Expr %T", typ))
	}
	return ""
}

func typeName(table *gosym.Table, fn *gosym.Func, s *gosym.Sym) string {
	// Get the argument position
	pos := -1
	for ii, v := range fn.Params {
		if v == s {
			pos = len(fn.Params) - ii - 1
			break
		}
	}
	if pos < 0 {
		return ""
	}
	file, _, _ := table.PCToLine(fn.Entry)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err == nil {
		var receiver string
		dot := strings.IndexByte(fn.Name, '.')
		name := fn.Name[dot+1:]
		pkg := fn.Name[:dot]
		if s := strings.LastIndex(pkg, "/"); s >= 0 {
			pkg = pkg[s+1:]
		}
		if ndot := strings.IndexByte(name, '.'); ndot >= 0 {
			receiver = strings.Trim(name[:ndot], "()")
			name = name[ndot+1:]
		}
		var fun *ast.FuncDecl
		ast.Inspect(f, func(n ast.Node) bool {
			if af, ok := n.(*ast.FuncDecl); ok && af.Name.Name == name {
				if receiver == "" || (af.Recv != nil && astTypeName(af.Recv.List[0].Type, "") == receiver) {
					fun = af
					return false
				}
			}
			return true
		})
		if fun != nil {
			ii := 0
			if receiver != "" {
				if pos == 0 {
					return astTypeName(fun.Recv.List[0].Type, pkg)
				}
				ii++
			}
			for _, v := range fun.Type.Params.List {
				c := len(v.Names)
				if ii+c > pos {
					return astTypeName(v.Type, pkg)
				}
				ii += c
			}
		}
	}
	return ""
}

func isInterface(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, tn string) bool {
	// TODO: Use go/build to parse the package and look for the type definution
	return false
}
