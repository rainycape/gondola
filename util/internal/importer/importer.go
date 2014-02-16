// Package importer implements an importer for go/types which
// imports packages from gc compiled objects and falls back
// to importing go code.
package importer

import (
	"code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"
	"go/build"
	"path/filepath"
)

type Importer struct {
	imports map[string]*types.Package
	conf    types.Config
}

func New() *Importer {
	return &Importer{
		imports: make(map[string]*types.Package),
		conf: types.Config{
			FakeImportC: true,
			Error:       errorHandler,
		},
	}
}

func (imp *Importer) Import(imports map[string]*types.Package, path string) (*types.Package, error) {
	if pkg := imp.imports[path]; pkg != nil {
		return pkg, nil
	}
	pkg, err := gcimporter.Import(imports, path)
	if err == nil {
		imports[path] = pkg
		imp.imports[path] = pkg
		return pkg, nil
	}
	bpkg, err := build.Import(path, ".", 0)
	if err != nil {
		return nil, err
	}
	var gofiles []string
	for _, v := range bpkg.GoFiles {
		gofiles = append(gofiles, filepath.Join(bpkg.Dir, v))
	}
	for _, v := range bpkg.CgoFiles {
		gofiles = append(gofiles, filepath.Join(bpkg.Dir, v))
	}
	conf := imp.conf
	conf.Import = imp.Import
	loader := &loader.Config{
		TypeChecker: conf,
		TypeCheckFuncBodies: func(name string) bool {
			// the parser fails to parse isatty
			return name != "gnd.la/log"
		},
		Build: &build.Default,
	}
	name := path
	if bpkg.Name == "main" {
		name = "main"
	}
	err = loader.CreateFromFilenames(name, gofiles...)
	if err != nil {
		return nil, err
	}
	if err := importImports(loader, imports, bpkg.Imports); err != nil {
		return nil, err
	}
	pr, err := loader.Load()
	if err != nil {
		return nil, err
	}
	pkg = pr.Created[0].Pkg
	imports[path] = pkg
	imp.imports[path] = pkg
	return pkg, nil
}

func importImports(cfg *loader.Config, imports map[string]*types.Package, req []string) error {
	for _, v := range req {
		if v == "C" || v == "unsafe" {
			// Pseudo-packages
			continue
		}
		if _, ok := cfg.ImportPkgs[v]; ok || imports[v] != nil {
			// Already imported
			continue
		}
		cfg.Import(v)
		bpkg, err := build.Import(v, ".", 0)
		if err != nil {
			return err
		}
		if err := importImports(cfg, imports, bpkg.Imports); err != nil {
			return err
		}
	}
	return nil
}

func errorHandler(err error) {}
