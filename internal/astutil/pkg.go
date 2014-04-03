package astutil

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
)

func New(p *build.Package, mode parser.Mode) (*token.FileSet, *ast.Package, error) {
	fset := token.NewFileSet()
	var names []string
	names = append(names, p.GoFiles...)
	names = append(names, p.CgoFiles...)
	files, err := ParseFiles(fset, p.Dir, names, mode)
	if err != nil {
		return nil, nil, err
	}
	a, _ := ast.NewPackage(fset, files, Importer, nil)
	return fset, a, nil
}
