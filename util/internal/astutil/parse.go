package astutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

func ParseFiles(fset *token.FileSet, abspath string, names []string, mode parser.Mode) (map[string]*ast.File, error) {
	files := make(map[string]*ast.File)
	for _, f := range names {
		absname := filepath.Join(abspath, f)
		file, err := parser.ParseFile(fset, absname, nil, mode)
		if err != nil {
			// Just ignore this file
			continue
		}
		files[absname] = file
	}
	return files, nil
}
