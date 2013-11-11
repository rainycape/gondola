package genutil

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Package represents a parsed package with all its
// dependencies. Use NewPackage to create a Package.
type Package struct {
	*types.Package
	dir string
}

// Dir returns the package source directory.
func (p *Package) Dir() string {
	return p.dir
}

func (p *Package) types(exported bool, include *regexp.Regexp, exclude *regexp.Regexp) []*types.Named {
	var packageTypes []*types.Named
	scope := p.Scope()
	prefix := p.Name() + "."
	for _, v := range scope.Names() {
		obj := scope.Lookup(v)
		if exported && !obj.IsExported() {
			continue
		}
		if _, ok := obj.(*types.Const); ok {
			continue
		}
		if _, ok := obj.(*types.Var); ok {
			continue
		}
		if named, ok := obj.Type().(*types.Named); ok {
			name := named.String()
			if strings.HasPrefix(name, prefix) {
				local := name[len(prefix):]
				if exclude != nil && exclude.MatchString(local) {
					continue
				}
				if include != nil && !include.MatchString(local) {
					continue
				}
				packageTypes = append(packageTypes, named)
			}
		}
	}
	return packageTypes
}

// Types returns the types declared in the package which match the required constraints.
// If excluded != nil, any type matching it gets excluded. If include != nil, only types
// matching it are returned.
func (p *Package) Types(include *regexp.Regexp, exclude *regexp.Regexp) []*types.Named {
	return p.types(false, include, exclude)
}

// ExportedTypes works like Types, but only returns types that are exported.
func (p *Package) ExportedTypes(include *regexp.Regexp, exclude *regexp.Regexp) []*types.Named {
	return p.types(true, include, exclude)
}

type _package struct {
	Path     string
	fset     *token.FileSet
	astFiles []*ast.File
	files    map[string]*file
}

type file struct {
	fset  *token.FileSet
	name  string
	ast   *ast.File
	lines [][]byte
}

// NewPackage returns a new Package struct, which can be
// used to generate code related to the package. The package
// might be given as either an absolute path or an import path.
// If the package can't be found or the package is not compilable,
// this function returns an error.
func NewPackage(path string) (*Package, error) {
	p := &_package{Path: path, fset: token.NewFileSet()}
	pkg, err := findPackage(path)
	if err != nil {
		return nil, fmt.Errorf("could not find package: %s", err)
	}
	fileNames := packageFiles(pkg)
	if len(fileNames) == 0 {
		return nil, fmt.Errorf("no go files")
	}
	p.astFiles = make([]*ast.File, len(fileNames))
	p.files = make(map[string]*file, len(fileNames))

	for ii, v := range fileNames {
		f, err := parseFile(p.fset, v)
		if err != nil {
			return nil, fmt.Errorf("could not parse %s: %s", v, err)
		}
		p.files[v] = f
		p.astFiles[ii] = f.ast
	}
	imp := newImporter()
	imp.Config.UseGcFallback = true
	context := types.Config{
		Import:      imp.Import,
		FakeImportC: true,
	}
	ipath := pkg.ImportPath
	if ipath == "." {
		// Check won't accept a "." import
		abs, err := filepath.Abs(pkg.Dir)
		if err != nil {
			return nil, err
		}
		for _, v := range strings.Split(build.Default.GOPATH, ":") {
			src := filepath.Join(v, "src")
			if strings.HasPrefix(abs, src) {
				ipath = abs[len(src)+1:]
				break
			}
		}
	}
	tpkg, err := context.Check(ipath, p.fset, p.astFiles, nil)
	if err != nil {
		// This error is caused by using fields in C structs, ignore it
		if !strings.Contains(err.Error(), "invalid type") {
			return nil, fmt.Errorf("error checking package: %s", err)
		}
	}
	return &Package{
		Package: tpkg,
		dir:     pkg.Dir,
	}, nil
}

func findPackage(path string) (*build.Package, error) {
	ctx := build.Default
	ctx.CgoEnabled = true
	p, err := ctx.Import(path, ".", 0)
	if err == nil {
		return p, err
	}
	return ctx.ImportDir(path, 0)
}

func packageFiles(pkg *build.Package) []string {
	var files []string
	for _, v := range pkg.GoFiles {
		files = append(files, filepath.Join(pkg.Dir, v))
	}
	for _, v := range pkg.CgoFiles {
		files = append(files, filepath.Join(pkg.Dir, v))
	}
	return files
}

func parseFile(fset *token.FileSet, fileName string) (f *file, err error) {
	rd, err := os.Open(fileName)
	if err != nil {
		return f, err
	}
	defer rd.Close()

	data, err := ioutil.ReadAll(rd)
	if err != nil {
		return f, err
	}

	astFile, err := parser.ParseFile(fset, fileName, bytes.NewReader(data), parser.ParseComments)
	if err != nil {
		return f, fmt.Errorf("could not parse: %s", err)
	}

	lines := bytes.Split(data, []byte("\n"))
	f = &file{fset: fset, name: fileName, ast: astFile, lines: lines}
	return f, nil
}
