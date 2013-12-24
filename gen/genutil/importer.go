package genutil

import (
	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/go.tools/importer"
	"go/build"
)

type genImporter struct {
	imports  map[string]*types.Package
	importer *importer.Importer
}

func newImporter(conf types.Config) *genImporter {
	config := &importer.Config{
		TypeChecker: conf,
		TypeCheckFuncBodies: func(_ string) bool {
			return false
		},
		Build: &build.Default,
	}
	imp := importer.New(config)
	return &genImporter{
		imports:  make(map[string]*types.Package),
		importer: imp,
	}
}

func (imp *genImporter) Import(imports map[string]*types.Package, path string) (*types.Package, error) {
	if pkg := imp.imports[path]; pkg != nil {
		return pkg, nil
	}
	bpkg, err := build.Import(path, ".", 0)
	if err != nil {
		return nil, err
	}
	var gofiles []string
	gofiles = append(gofiles, bpkg.GoFiles...)
	gofiles = append(gofiles, bpkg.CgoFiles...)
	files, err := importer.ParseFiles(imp.importer.Fset, bpkg.Dir, gofiles...)
	if err != nil {
		return nil, err
	}
	name := path
	if bpkg.Name == "main" {
		name = "main"
	}
	pkg := imp.importer.CreatePackage(name, files...).Pkg
	ppath := pkg.Path()
	imports[ppath] = pkg
	imp.imports[ppath] = pkg
	return pkg, nil
}
