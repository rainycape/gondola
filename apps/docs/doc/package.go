package doc

import (
	"bufio"
	"bytes"
	"fmt"
	"gnd.la/app"
	"gnd.la/apps/docs/doc/printer"
	"gnd.la/internal/astutil"
	"gnd.la/internal/pkgutil"
	"gnd.la/util/generic"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	Context           = build.Default
	App               *app.App
	SourceHandlerName string
	DocHandlerName    string
	valueRe           = regexp.MustCompile("([A-Z]\\w+)((?:\\s+<a.*?</a>)?\\s+=)")
	httpRe            = regexp.MustCompile("(https?://.*?)(\\s|\\.($|\\s|<|>))")
)

type ImportOptions struct {
	Shallow bool
}

func noBuildable(err error) bool {
	return strings.Contains(err.Error(), "no buildable")
}

func buildContext() *build.Context {
	return &Context
}

type Package struct {
	fset          *token.FileSet
	name          string
	dir           string
	bpkg          *build.Package
	apkg          *ast.Package
	dpkg          *doc.Package
	bodies        map[*ast.FuncDecl]*ast.BlockStmt // go/doc strips function bodies
	Packages      []*Package
	examples      []*Example
	examplesByKey map[string][]*Example
}

func (p *Package) symbolHref(symbol string) string {
	if p.apkg == nil || p.apkg.Scope == nil {
		return ""
	}
	key := symbol
	if key[len(key)-1] == ')' && key[len(key)-2] == '(' {
		key = key[:len(key)-2]
	}
	if key[len(key)-1] == '.' {
		key = key[:len(key)-1]
	}
	if obj := p.apkg.Scope.Objects[key]; obj != nil {
		switch obj.Kind {
		case ast.Typ:
			return "#" + TypeId(key)
		case ast.Fun:
			return "#" + FuncId(key)
		case ast.Con:
			return "#" + ConstId(key)
		case ast.Var:
			return "#" + VarId(key)
		}
	}
	if dot := strings.IndexByte(key, '.'); dot > 0 {
		tn := key[:dot]
		fn := key[dot+1:]
		if obj := p.apkg.Scope.Objects[tn]; obj != nil && obj.Kind == ast.Typ {
			for _, v := range p.dpkg.Types {
				if v.Name == tn {
					for _, m := range v.Methods {
						if m.Name == fn {
							return "#" + MethodId(tn, fn)
						}
					}
					return ""
				}
			}
		}
	}
	return ""
}

func (p *Package) href(word string, scope string) string {
	slash := strings.IndexByte(word, '/')
	dot := strings.IndexByte(word, '.')
	if slash > 0 || dot > 0 {
		// Check if there's a type or function mentioned
		// after the package.
		if pn, tn := pkgutil.SplitQualifiedName(word); pn != "" && tn != "" {
			if pn[0] == '*' {
				pn = pn[1:]
			}
			if pkg, err := ImportPackage(pn); err == nil {
				if sr := pkg.symbolHref(tn); sr != "" {
					return App.MustReverse(DocHandlerName, pn) + sr
				}
			}
			if pn == p.dpkg.Name {
				return p.symbolHref(tn)
			}
		} else if _, err := buildContext().Import(word, "", build.FindOnly); err == nil {
			return App.MustReverse(DocHandlerName, word)
		}
	}
	if dot > 0 {
		// Check the package imports, to see if any of them matches
		// TODO: Check for packages imported with a different local
		// name.
		base := word[:dot]
		for _, v := range p.bpkg.Imports {
			if path.Base(v) == base && v != base {
				return p.href(v+"."+word[dot+1:], scope)
			}
		}
	}
	if word[0]&0x20 == 0 {
		// Uppercase
		if scope != "" {
			if href := p.symbolHref(scope + "." + word); href != "" {
				return href
			}
		}
		return p.symbolHref(word)
	}
	return ""
}

func (p *Package) LinkType(x string, sel string) string {
	if x != "" {
		sel = x + "." + sel
	}
	return p.href(sel, "")
}

func (p *Package) Linkify(comment string, group *ast.CommentGroup) string {
	pr := ""
	if group != nil && len(group.List) > 0 && group.List[0].Text == comment {
		if len(comment) > 3 {
			if sp := strings.IndexByte(comment[3:], ' '); sp >= 0 {
				pr = comment[:sp]
				comment = comment[sp:]
			}
		}
	}
	var buf bytes.Buffer
	p.linkify(&buf, comment, "", nil)
	return pr + buf.String()
}

func (p *Package) writeWord(bw *bufio.Writer, buf *bytes.Buffer, scope string, ignored map[string]struct{}) {
	if word := buf.String(); word != "" {
		if _, ign := ignored[word]; ign {
			bw.WriteString(word)
		} else {
			if href := p.href(word, scope); href != "" {
				bw.WriteString("<a href=\"")
				bw.WriteString(href)
				bw.WriteString("\">")
				bw.WriteString(word)
				bw.WriteString("</a>")
			} else {
				bw.WriteString(word)
			}
		}
	}
}

func (p *Package) linkify(w io.Writer, input string, scope string, ignored map[string]struct{}) error {
	bw := bufio.NewWriterSize(w, 512)
	var buf bytes.Buffer
	for ii := 0; ii < len(input); ii++ {
		c := input[ii]
		switch c {
		// Include * and & in the list of stop characters,
		// so pointers get the link for the pointed type.
		// Include ;, so escaped amperstands do not end up
		// in the type names.
		case ',', ' ', '\n', '\t', '(', ')', '*', '&', '{', '}', ';', '<', '>':
			p.writeWord(bw, &buf, scope, ignored)
			bw.WriteByte(c)
			buf.Reset()
		case '.':
			if next := ii + 1; next < len(input) {
				if nc := input[next]; nc == ' ' || nc == '\t' || nc == '\n' {
					p.writeWord(bw, &buf, scope, ignored)
					bw.WriteByte(c)
					buf.Reset()
					continue
				}
			}
			fallthrough
		default:
			buf.WriteByte(c)
		}
	}
	p.writeWord(bw, &buf, scope, ignored)
	return bw.Flush()
}

func (p *Package) FileSet() *token.FileSet {
	return p.fset
}

func (p *Package) File(name string) *ast.File {
	if p.apkg != nil {
		return p.apkg.Files[filepath.Join(p.bpkg.Dir, name)]
	}
	return nil
}

func (p *Package) NewAST() (*token.FileSet, *ast.Package, error) {
	if p.bpkg == nil {
		return nil, nil, errInvalidPackage
	}
	return astutil.New(p.bpkg, 0)
}

func (p *Package) Name() string {
	if p.bpkg != nil {
		return p.bpkg.Name
	}
	return p.name
}

func (p *Package) Dir() string {
	if p.bpkg != nil {
		return p.bpkg.Dir
	}
	return p.dir
}

func (p *Package) ImportPath() string {
	if p.bpkg != nil {
		path := p.bpkg.ImportPath
		if path == "." {
			if gr := buildContext().GOROOT; strings.HasPrefix(p.bpkg.Dir, gr) {
				path = p.bpkg.Dir[len(gr)+5:]
			}
		}
		if gp := buildContext().GOPATH; strings.HasPrefix(path, gp) {
			// Skip src/ after GOPATH
			path = path[len(gp)+4:]
		}
		return path
	}
	return ""
}

func (p *Package) IsStd() bool {
	return strings.HasPrefix(p.Dir(), buildContext().GOROOT+string(filepath.Separator))
}

func (p *Package) IsMain() bool {
	return p.bpkg != nil && p.dpkg.Name == "main"
}

func (p *Package) IsEmpty() bool {
	return p.bpkg == nil
}

func (p *Package) Imports() []string {
	if p.bpkg == nil {
		return nil
	}
	var imports []string
	imports = append(imports, p.bpkg.Imports...)
	imports = append(imports, p.bpkg.TestImports...)
	return imports
}

func (p *Package) NonStdImports() []string {
	var nonstd []string
	for _, v := range p.Imports() {
		if strings.IndexByte(v, '.') >= 0 {
			nonstd = append(nonstd, v)
		}
	}
	return nonstd
}

func (p *Package) CommandName() string {
	if p.IsMain() {
		return path.Base(p.ImportPath())
	}
	return ""
}

func (p *Package) Synopsis() string {
	if p.dpkg != nil {
		return doc.Synopsis(p.dpkg.Doc)
	}
	return ""
}

func (p *Package) Stats() (*Stats, error) {
	return NewStats(p)
}

func (p *Package) Filenames() []string {
	if b := p.bpkg; b != nil {
		return pkgutil.Filenames(b)
	}
	return nil
}

func (p *Package) GoFilenames() []string {
	if b := p.bpkg; b != nil {
		var files []string
		for _, v := range [][]string{b.GoFiles, b.CgoFiles} {
			files = append(files, v...)
		}
		sort.Strings(files)
		return files
	}
	return nil
}

func (p *Package) LineCount() (int, error) {
	if b := p.bpkg; b != nil {
		return pkgutil.LineCount(p.bpkg)
	}
	return 0, errInvalidPackage
}

func (p *Package) ReversePos(n ast.Node) string {
	pos := p.fset.Position(n.Pos())
	return p.ReversePosition(pos)
}

func (p *Package) ReversePosition(pos token.Position) string {
	return p.ReverseFilenameLine(pos.Filename, pos.Line)
}

func (p *Package) ReverseFilenameLine(filename string, line int) string {
	return fmt.Sprintf("%s#line-%d", p.ReverseFilename(filename), line)
}

func (p *Package) ReverseFilename(filename string) string {
	filename = path.Base(filename)
	rel := path.Join(p.ImportPath(), filename)
	return App.MustReverse(SourceHandlerName, rel)
}

func (p *Package) FuncLink(fn *ast.FuncDecl) string {
	recv := FuncReceiver(fn)
	if ast.IsExported(fn.Name.Name) && (recv == "" || ast.IsExported(recv)) {
		if recv != "" {
			return "#" + MethodId(recv, fn.Name.Name)
		}
		return "#" + FuncId(fn.Name.Name)
	}
	return p.ReversePos(fn)
}

func (p *Package) HasDoc() bool {
	if p.dpkg != nil {
		synopsis := strings.TrimSpace(doc.Synopsis(p.dpkg.Doc))
		full := strings.Replace(strings.Replace(strings.TrimSpace(p.dpkg.Doc), "\r", "", -1), "\n", " ", -1)
		return synopsis != full
	}
	return false
}

func (p *Package) Doc() *doc.Package {
	return p.dpkg
}

func (p *Package) Funcs() []*doc.Func {
	var funcs []*doc.Func
	funcs = append(funcs, p.dpkg.Funcs...)
	for _, v := range p.dpkg.Types {
		funcs = append(funcs, v.Funcs...)
	}
	generic.Sort(funcs, "Name")
	return funcs
}

func (p *Package) FuncExamples(fn *doc.Func) []*Example {
	if p.examples == nil {
		p.Examples()
	}
	key := fn.Name
	if fn.Recv != "" {
		recv := fn.Recv
		if recv[0] == '*' {
			recv = recv[1:]
		}
		key = recv + "." + key
	}
	return p.examplesByKey[key]
}

func (p *Package) Examples() []*Example {
	if p.examples != nil || p.bpkg == nil {
		return p.examples
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, val := range [][]string{p.bpkg.TestGoFiles, p.bpkg.XTestGoFiles} {
		for _, v := range val {
			f, err := parser.ParseFile(fset, filepath.Join(p.Dir(), v), nil, parser.ParseComments)
			if err == nil {
				files = append(files, f)
			}
		}
	}
	for _, v := range doc.Examples(files...) {
		p.examples = append(p.examples, &Example{
			fset:    fset,
			pkg:     p,
			example: v,
		})
	}
	p.examplesByKey = make(map[string][]*Example)
	for _, v := range p.examples {
		k := v.Key()
		p.examplesByKey[k] = append(p.examplesByKey[k], v)
	}
	if p.examples == nil {
		p.examples = make([]*Example, 0, 1)
	}
	return p.examples
}

func (p *Package) html(text string, scope string, ignored map[string]struct{}) template.HTML {
	var buf bytes.Buffer
	doc.ToHTML(&buf, text, nil)
	var out bytes.Buffer
	p.linkify(&out, buf.String(), scope, ignored)
	return template.HTML(out.String())
}

func (p *Package) HTML(text string) template.HTML {
	p.html(text, "", nil)
	return p.html(text, "", nil)
}

func (p *Package) scopeParameters(node interface{}) (string, map[string]struct{}) {
	var scope string
	var ignored map[string]struct{}
	ignore := func(x string) {
		if ignored == nil {
			ignored = make(map[string]struct{})
		}
		ignored[x] = struct{}{}
	}
	switch n := node.(type) {
	case *ast.FuncDecl:
		ignore(n.Name.Name)
		if n.Recv != nil {
			scope = astutil.Ident(n.Recv.List[0].Type)
			if scope != "" && scope[0] == '*' {
				scope = scope[1:]
			}
		}
	case *ast.GenDecl:
		for _, spec := range n.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				scope = s.Name.Name
				ignore(scope)
			case *ast.ValueSpec:
				for _, name := range s.Names {
					ignore(name.Name)
				}
			}
		}
	}
	return scope, ignored
}

func (p *Package) ScopedHTML(text string, scope interface{}) template.HTML {
	// TODO: When linkifying a method doc, linkify references to other
	// methods in the form <receiver_var_name>.method. See
	// gnd.la/util/semver for an example.
	name, ignored := p.scopeParameters(scope)
	return p.html(text, name, ignored)
}

func (p *Package) HTMLDoc() template.HTML {
	return p.HTML(p.dpkg.Doc)
}

func (p *Package) HTMLDecl(node interface{}) (template.HTML, error) {
	cfg := printer.Config{
		HTML:     true,
		Tabwidth: 8,
		Linker:   p,
	}
	var buf bytes.Buffer
	err := cfg.Fprint(&buf, p.fset, node)
	s := buf.String()
	if strings.HasPrefix(s, "<span class=\"token\">const</span>") {
		s = valueRe.ReplaceAllString(s, "<span id=\""+constPrefix+"${1}\">${1}</span>${2}")
	} else if strings.HasPrefix(s, "<span class=\"token\">var</span>") {
		s = valueRe.ReplaceAllString(s, "<span id=\""+varPrefix+"${1}\">${1}</span>${2}")
	}
	return template.HTML(s), err
}

func importBuildPackage(p string) (*build.Package, error) {
	bctx := buildContext()
	b, err := bctx.Import(p, "", 0)
	if err != nil && !noBuildable(err) {
		b, err = bctx.ImportDir(p, 0)
		if err != nil && !noBuildable(err) {
			// Go standard command?
			cmdDir := filepath.Join(bctx.GOROOT, "src", filepath.FromSlash(p))
			b, err = bctx.ImportDir(cmdDir, 0)
		}
	}
	return b, err
}

func importSubpackages(p string) (string, []*Package, error) {
	if !filepath.IsAbs(p) {
		dir := filepath.Join(buildContext().GOPATH, "src", p)
		if s, err := ImportPackages(dir); err == nil {
			return dir, s, nil
		}
		dir = filepath.Join(buildContext().GOROOT, "src", "pkg", p)
		if s, err := ImportPackages(dir); err == nil {
			return dir, s, nil
		}
		dir = filepath.Join(buildContext().GOROOT, "src", p)
		if s, err := ImportPackages(dir); err == nil {
			return dir, s, nil
		}
	}
	sub, err := ImportPackages(p)
	return p, sub, err
}

func ImportPackage(p string) (*Package, error) {
	return importPackage(p, false)
}

func ImportPackageOpts(p string, opts *ImportOptions) (*Package, error) {
	shallow := false
	if opts != nil {
		shallow = opts.Shallow
	}
	return importPackage(p, shallow)
}

func importPackage(p string, shallow bool) (*Package, error) {
	b, err := importBuildPackage(p)
	if err != nil {
		if noBuildable(err) && !shallow {
			dir, sub, err := importSubpackages(p)
			if err != nil {
				return nil, err
			}
			return &Package{name: path.Base(p), dir: dir, Packages: sub}, nil

		}
	}
	fset := token.NewFileSet()
	var names []string
	names = append(names, b.GoFiles...)
	names = append(names, b.CgoFiles...)
	files, err := astutil.ParseFiles(fset, b.Dir, names, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	bodies := make(map[*ast.FuncDecl]*ast.BlockStmt)
	for _, f := range files {
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				bodies[fn] = fn.Body
			}
		}
	}
	// NewPackage will always return errors because it won't
	// resolve builtin types.
	a, _ := ast.NewPackage(fset, files, astutil.Importer, nil)
	flags := doc.AllMethods
	if p == "builtin" {
		flags |= doc.AllDecls
	}
	pkg := &Package{
		fset:   fset,
		bpkg:   b,
		apkg:   a,
		dpkg:   doc.New(a, b.ImportPath, flags),
		bodies: bodies,
	}
	if !shallow {
		sub, err := ImportPackages(b.Dir)
		if err != nil {
			return nil, err
		}
		pkg.Packages = sub
	}
	return pkg, nil
}

func ImportPackages(dir string) ([]*Package, error) {
	var pkgs []*Package
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, v := range files {
		n := v.Name()
		if n == "test_data" || n == "testdata" || n[0] == '.' || n[0] == '_' {
			continue
		}
		abs := filepath.Join(dir, n)
		// Follow symlinks
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			pkg, err := ImportPackage(abs)
			if err != nil {
				if noBuildable(err) {
					sub, err := ImportPackages(abs)
					if err != nil {
						return nil, err
					}
					pkgs = append(pkgs, sub...)
					continue
				}
				return nil, err
			}
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs, nil
}
