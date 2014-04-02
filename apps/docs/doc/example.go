package doc

import (
	"bytes"
	"gnd.la/apps/docs/doc/printer"
	"go/ast"
	"go/doc"
	"go/token"
	"html/template"
	"strings"
)

type Example struct {
	fset    *token.FileSet
	pkg     *Package
	example *doc.Example
}

func (e *Example) Id() string {
	return "example-" + e.example.Name
}

func (e *Example) Key() string {
	title := e.Title()
	if !strings.Contains(title, " ") {
		return title
	}
	key := e.Name()
	if p := strings.IndexByte(key, '_'); p >= 0 {
		key = key[:p]
	}
	return key
}

func (e *Example) Name() string {
	return e.example.Name
}

func (e *Example) Title() string {
	name := e.Name()
	if p := strings.IndexByte(name, '_'); p >= 0 {
		suffix := name[p+1:]
		if ast.IsExported(suffix) {
			suffix = "." + suffix
		} else {
			suffix = " (" + strings.ToUpper(suffix[:1]) + suffix[1:] + ")"
		}
		name = name[:p] + suffix
	}
	return name
}

func (e *Example) EmptyOutput() bool {
	return e.example.EmptyOutput
}

func (e *Example) Output() string {
	return e.example.Output
}

func (e *Example) Doc() string {
	return e.example.Doc
}

func (e *Example) HTML() (template.HTML, error) {
	cfg := printer.Config{
		HTML:     true,
		Tabwidth: 8,
		Linker:   e.pkg,
	}
	var buf bytes.Buffer
	var node interface{}
	if e.example.Play != nil {
		node = e.example.Play
	} else {
		node = e.example.Code
		if bk, ok := node.(*ast.BlockStmt); ok {
			node = bk.List
		}
	}
	err := cfg.Fprint(&buf, e.fset, node)
	return template.HTML(buf.String()), err
}
