package astutil

import (
	"go/ast"
	"go/token"
	"strconv"
)

func Selector(expr ast.Expr) (x string, sel string) {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return Selector(e.X)
	case *ast.UnaryExpr:
		return Selector(e.X)
	case *ast.CompositeLit:
		return Selector(e.Type)
	case *ast.SelectorExpr:
		return Ident(e.X), Ident(e.Sel)
	}
	return "", ""
}

func Ident(node ast.Expr) string {
	switch n := node.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.SelectorExpr:
		x := Ident(n.X)
		s := Ident(n.Sel)
		if x != "" && s != "" {
			return x + "." + s
		}
		return s
	case *ast.StarExpr:
		return "*" + Ident(n.X)
	}
	return ""
}

func StringLiteral(f *token.FileSet, n ast.Expr) (string, *token.Position) {
	if lit, ok := n.(*ast.BasicLit); ok {
		if lit.Kind == token.STRING {
			unquoted, err := strconv.Unquote(lit.Value)
			if err == nil {
				pos := f.Position(lit.Pos())
				return unquoted, &pos
			}
		}
	}
	if bin, ok := n.(*ast.BinaryExpr); ok && bin.Op == token.ADD {
		x, xpos := StringLiteral(f, bin.X)
		y, ypos := StringLiteral(f, bin.Y)
		if xpos != nil && ypos != nil {
			return x + y, xpos
		}
	}
	return "", nil
}
