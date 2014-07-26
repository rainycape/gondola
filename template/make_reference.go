// +build NONE

// This file generates an article for the gondolaweb.com site, which
// documents the default template functions.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"code.google.com/p/go.tools/go/types"

	"gnd.la/apps/articles/article"
	"gnd.la/internal/astutil"
	"gnd.la/internal/gen/genutil"
)

func commentsBetween(f *ast.File, begin token.Pos, end token.Pos) string {
	var buf bytes.Buffer
	var ignore token.Pos
	ignoreUntilNextGroup := func(c *ast.Comment) {
		ignore = c.Slash + token.Pos(len(c.Text)) + 3 // allow for newlines, tabs, etc..
	}
	for _, group := range f.Comments {
		for _, v := range group.List {
			if v.Slash > begin && v.Slash < end {
				if v.Slash < ignore {
					ignoreUntilNextGroup(v)
				}
				text := strings.TrimSpace(strings.TrimPrefix(v.Text, "//"))
				if text != "" {
					if buf.Len() == 0 && text[0] == '!' {
						// Marked as non-doc comment
						ignoreUntilNextGroup(v)
						continue
					}
					if buf.Len() > 0 {
						buf.WriteByte(' ')
					}
					buf.WriteString(text)
				}
			}
		}
	}
	return buf.String()
}

type funcDoc struct {
	doc   string
	decl  string
	alias string
}

func documentFunction(f *ast.File, pkg *genutil.Package, value ast.Expr, prev token.Pos, cur token.Pos) *funcDoc {
	fileScope := pkg.Info().Scopes[f]
	val := pkg.Info().Types[value]
	ftyp := val.Type
	for {
		named, ok := ftyp.(*types.Named)
		if !ok {
			break
		}
		ftyp = named.Underlying()
	}
	if sel, ok := value.(*ast.SelectorExpr); ok {
		x := astutil.Ident(sel.X)
		obj := fileScope.Lookup(x)
		if pkgName, ok := obj.(*types.PkgName); ok {
			expr := astutil.Ident(sel.Sel)
			return &funcDoc{
				decl:  fmt.Sprintf("%s", ftyp),
				alias: pkgName.Pkg().Path() + "." + expr,
			}
		}
		return nil
	}
	if _, ok := value.(*ast.Ident); ok {
		if comments := commentsBetween(f, prev, cur); comments != "" {
			return &funcDoc{
				doc:  comments,
				decl: fmt.Sprintf("%s", ftyp),
			}
		}
		return nil
	}
	return nil
}

func main() {
	pkg, err := genutil.NewPackage("gnd.la/template")
	if err != nil {
		panic(err)
	}
	var art *article.Article
	if len(os.Args) > 1 {
		art, err = article.OpenFile(os.Args[1])
		if err != nil {
			panic(fmt.Errorf("error loading previous article from %s: %s", os.Args[1], err))
		}
	} else {
		art = &article.Article{}
	}
	abs, err := filepath.Abs("funcs.go")
	if err != nil {
		panic(err)
	}
	f := pkg.ASTFiles()[abs]
	var inMap bool
	var lastPos token.Pos
	docs := make(map[string]*funcDoc)
	ast.Inspect(f, func(n ast.Node) bool {
		if !inMap {
			if vs, ok := n.(*ast.ValueSpec); ok {
				for _, n := range vs.Names {
					if n.Name == "templateFuncs" {
						inMap = true
						lastPos = vs.Pos()
						break
					}
				}
			}
		}
		if inMap {
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if k := literalString(kv.Key); k != "" {
					if doc := documentFunction(f, pkg, kv.Value, lastPos, kv.Pos()); doc != nil {
						docs[k] = doc
					}
				}
				lastPos = kv.Pos()
			}
		}
		return true
	})
	// Write the markdown
	var buf bytes.Buffer
	chrs := make(map[string]byte)
	names := make([]string, 0, len(docs))
	for k := range docs {
		if k[0] == '#' || k[0] == '!' || k[0] == '@' {
			chr := k[0]
			k = k[1:]
			chrs[k] = chr
		}
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		chr := chrs[n]
		lookup := n
		if chr != 0 {
			lookup = string(chr) + n
		}
		doc := docs[lookup]
		fmt.Fprintf(&buf, "- **%s** *%s*", n, doc.decl)
		buf.WriteString("\n\n\t")
		if doc.alias != "" {
			fmt.Fprintf(&buf, "%s is an alias for [%s](/doc/pkg/%s)", n, doc.alias, linkFunc(doc.alias))
		} else {
			buf.WriteString(doc.doc)
		}
		buf.WriteString("\n\n")
	}
	art.Text = buf.Bytes()
	art.Set("updated", "now")
	var out bytes.Buffer
	if _, err := art.WriteTo(&out); err != nil {
		panic(err)
	}
	if art.Filename != "" {
		if err := ioutil.WriteFile(art.Filename, out.Bytes(), 0644); err != nil {
			panic(err)
		}
	} else {
		io.Copy(os.Stdout, &out)
	}
}

func linkFunc(name string) string {
	dot := strings.LastIndex(name, ".")
	if dot > 0 {
		return name[:dot] + "#func-" + name[dot+1:]
	}
	return name
}

func literalString(expr ast.Expr) string {
	if bl, ok := expr.(*ast.BasicLit); ok {
		if bl.Kind == token.STRING {
			unq, _ := strconv.Unquote(bl.Value)
			return unq
		}
	}
	return ""
}
