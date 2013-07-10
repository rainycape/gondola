package astutil

import (
	"go/ast"
	"go/token"
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

func Ident(n ast.Expr) string {
	if ident, ok := n.(*ast.Ident); ok {
		return ident.Name
	}
	if sel, ok := n.(*ast.SelectorExpr); ok {
		x := Ident(sel.X)
		s := Ident(sel.Sel)
		if x != "" && s != "" {
			return x + "." + s
		}
	}
	return ""
}

func StringLiteral(f *token.FileSet, n ast.Expr) (string, *token.Position) {
	if lit, ok := n.(*ast.BasicLit); ok {
		if lit.Kind == token.STRING {
			pos := f.Position(lit.Pos())
			return unquote(lit.Value), &pos
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
