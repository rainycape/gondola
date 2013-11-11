package genutil

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"honnef.co/go/importer"
	"io/ioutil"
	"os"
	"path/filepath"
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
	imp := importer.New()
	imp.Config.UseGcFallback = true
	context := types.Config{
		Import: imp.Import,
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
		return nil, fmt.Errorf("error checking package: %s", err)
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
	files := make([]string, len(pkg.GoFiles))
	for ii, v := range pkg.GoFiles {
		files[ii] = filepath.Join(pkg.Dir, v)
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
