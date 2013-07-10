package astutil

import (
	"go/ast"
	"path"
)

// Imports return whetever the given ast.File imports
// the package named pkg. It also returns its local
// imported name.
func Imports(f *ast.File, pkg string) (string, bool) {
	for _, v := range f.Imports {
		if v.Path.Value == "\""+pkg+"\"" {
			if v.Name != nil {
				if v.Name.Name == "." {
					return "", true
				}
				return v.Name.Name, true
			}
			return path.Base(pkg), true
		}
	}
	// TODO: Check if the pkg declared in f is pkg
	return "", false
}
