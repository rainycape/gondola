package messages

import (
	"fmt"
	"gnd.la/internal/astutil"
	"gnd.la/internal/pkgutil"
	"gnd.la/internal/templateutil"
	"gnd.la/log"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template/parse"
)

func DefaultFunctions() []*Function {
	return []*Function{
		// Singular functions without context
		{Name: "gnd.la/i18n.T", Start: 1},
		{Name: "gnd.la/i18n.Errorf"},
		{Name: "gnd.la/i18n.Sprintf", Start: 1},
		{Name: "gnd.la/i18n.NewError"},
		{Name: "gnd.la/app.Context.T", Start: 1},
		{Name: "t", Template: true},
		// Singular functions with context
		{Name: "gnd.la/i18n.Tc", Context: true, Start: 1},
		{Name: "gnd.la/i18n.Sprintfc", Context: true, Start: 1},
		{Name: "gnd.la/i18n.Errorfc", Context: true},
		{Name: "gnd.la/i18n.NewErrorc", Context: true},
		{Name: "gnd.la/app.Context.Tc", Context: true, Start: 1},
		{Name: "tc", Template: true, Context: true},
		// Plural functions without context
		{Name: "gnd.la/i18n.Tn", Plural: true, Start: 1},
		{Name: "gnd.la/i18n.Sprintfn", Plural: true, Start: 1},
		{Name: "gnd.la/i18n.Errorfn", Plural: true},
		{Name: "gnd.la/i18n.NewErrorn", Plural: true},
		{Name: "gnd.la/app.Context.Tn", Plural: true, Start: 1},
		{Name: "tn", Template: true, Plural: true},
		// Plural functions with context
		{Name: "gnd.la/i18n.Tnc", Context: true, Plural: true, Start: 1},
		{Name: "gnd.la/i18n.Errorfnc", Context: true, Plural: true},
		{Name: "gnd.la/i18n.Sprintfnc", Context: true, Plural: true, Start: 1},
		{Name: "gnd.la/i18n.NewErrornc", Context: true, Plural: true},
		{Name: "gnd.la/app.Context.Tnc", Context: true, Plural: true, Start: 1},
		{Name: "tnc", Template: true, Context: true, Plural: true},
	}
}

func DefaultTypes() []string {
	return []string{
		"gnd.la/i18n.String",
	}
}

func DefaultTagFields() []string {
	return []string{
		"help",
		"label",
		"placeholder",
	}
}

type ExtractOptions struct {
	Functions []*Function
	Types     []string
	TagFields []string
}

func DefaultExtractOptions() *ExtractOptions {
	return &ExtractOptions{
		Functions: DefaultFunctions(),
		Types:     DefaultTypes(),
		TagFields: DefaultTagFields(),
	}
}

func Extract(dir string, opts *ExtractOptions) ([]*Message, error) {
	if opts == nil {
		opts = DefaultExtractOptions()
	}
	messages := make(messageMap)
	err := extract(messages, dir, opts)
	if err != nil {
		return nil, err
	}
	return messages.Messages(), nil
}

func extract(messages messageMap, dir string, opts *ExtractOptions) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	infos, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	for _, v := range infos {
		name := v.Name()
		p := filepath.Join(dir, name)
		if v.IsDir() {
			if !pkgutil.IsPackage(p) {
				if err := extract(messages, p, opts); err != nil {
					return err
				}
			}
			continue
		}
		switch strings.ToLower(filepath.Ext(name)) {
		// TODO: text and strings files
		case ".html", ".txt":
			if err := extractTemplateMessages(messages, p, opts); err != nil {
				return err
			}
		case ".go":
			if err := extractGoMessages(messages, p, opts); err != nil {
				return err
			}
		case ".po", ".pot":
			// Do nothing
		}
	}
	return nil
}

func extractGoMessages(messages messageMap, path string, opts *ExtractOptions) error {
	log.Debugf("Extracting messages from Go file %s", path)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing go file %s: %s", path, err)
	}
	if opts != nil {
		for _, v := range opts.Functions {
			if v.Template {
				continue
			}
			if err := extractGoFunc(messages, fset, f, v); err != nil {
				return err
			}
		}
		for _, v := range opts.Types {
			if err := extractGoType(messages, fset, f, v); err != nil {
				return err
			}
		}
		for _, v := range opts.TagFields {
			if err := extractGoTagField(messages, fset, f, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractGoFunc(messages messageMap, fset *token.FileSet, f *ast.File, fn *Function) error {
	calls, err := astutil.Calls(fset, f, fn.Name)
	if err != nil {
		return err
	}
	n := fn.Start
	if fn.Context {
		n++
	}
	var message *Message
	var position *token.Position
	for _, c := range calls {
		if fn.Plural {
			if len(c.Args) < n+3 {
				log.Debugf("Skipping plural function %s (%v) - not enough arguments", astutil.Ident(c.Fun), fset.Position(c.Pos()))
				continue
			}
			slit, spos := astutil.StringLiteral(fset, c.Args[n])
			if slit == "" || spos == nil {
				log.Debugf("Skipping first argument to plural function %s (%v) - not a literal", astutil.Ident(c.Fun), fset.Position(c.Pos()))
				continue
			}
			plit, ppos := astutil.StringLiteral(fset, c.Args[n+1])
			if plit == "" || ppos == nil {
				log.Debugf("Skipping second argument to plural function %s (%v) - not a literal", astutil.Ident(c.Fun), fset.Position(c.Pos()))
				continue
			}
			message = &Message{
				Singular: slit,
				Plural:   plit,
			}
			position = spos
		} else {
			if len(c.Args) < n+1 {
				log.Debugf("Skipping singular function %s (%v) - not enough arguments", astutil.Ident(c.Fun), fset.Position(c.Pos()))
				continue
			}
			lit, pos := astutil.StringLiteral(fset, c.Args[n])
			if lit == "" || pos == nil {
				log.Debugf("Skipping argument to singular function %s (%v) - not a literal", astutil.Ident(c.Fun), fset.Position(c.Pos()))
				continue
			}
			message = &Message{
				Singular: lit,
			}
			position = pos
		}
		if message != nil && position != nil {
			if fn.Context {
				ctx, cpos := astutil.StringLiteral(fset, c.Args[fn.Start])
				if ctx == "" || cpos == nil {
					log.Debugf("Skipping argument to context function %s (%v) - empty context", astutil.Ident(c.Fun), fset.Position(c.Pos()))
					continue
				}
				message.Context = ctx
			}
			if err := messages.Add(message, position, comments(fset, f, position)); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractGoType(messages messageMap, fset *token.FileSet, f *ast.File, typ string) error {
	// for castings
	calls, err := astutil.Calls(fset, f, typ)
	if err != nil {
		return err
	}
	for _, c := range calls {
		if len(c.Args) > 0 {
			lit, pos := astutil.StringLiteral(fset, c.Args[0])
			if pos == nil {
				p := fset.Position(c.Pos())
				log.Debugf("Skipping cast to %s (%v) - not a literal", typ, p)
				continue
			}
			comment := comments(fset, f, pos)
			if err := messages.AddString(&astutil.String{Value: lit, Position: pos}, comment); err != nil {
				return err
			}
		}
	}
	strings, err := astutil.Strings(fset, f, typ)
	if err != nil {
		return err
	}
	for _, s := range strings {
		comment := comments(fset, f, s.Position)
		if err := messages.AddString(s, comment); err != nil {
			return err
		}
	}
	return nil
}

func extractGoTagField(messages messageMap, fset *token.FileSet, f *ast.File, tagField string) error {
	strings, err := astutil.TagFields(fset, f, tagField)
	if err != nil {
		return err
	}
	for _, s := range strings {
		comment := comments(fset, f, s.Position)
		if err := messages.AddString(s, comment); err != nil {
			return err
		}
	}
	return nil
}

func extractTemplateMessages(messages messageMap, path string, opts *ExtractOptions) error {
	log.Debugf("Extracting messages from template file %s", path)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	funcs := make(map[string]*Function)
	if opts != nil {
		for _, v := range opts.Functions {
			if v.Template {
				funcs[v.Name] = v
			}
		}
	}
	text := string(b)
	treeSet, err := templateutil.Parse(path, text)
	if err != nil {
		return err
	}
	for _, v := range treeSet {
		if err := templateutil.ReplaceTranslatableBlocks(v, "t"); err != nil {
			return err
		}
		templateutil.WalkTree(v, func(n, p parse.Node) {
			var fname string
			switch n.Type() {
			case parse.NodeIdentifier:
				fname = n.(*parse.IdentifierNode).Ident
			case parse.NodeField:
				ident := n.(*parse.FieldNode).Ident
				if len(ident) > 1 {
					fname = ident[len(ident)-1]
				}
			case parse.NodeVariable:
				ident := n.(*parse.VariableNode).Ident
				if len(ident) > 1 {
					fname = ident[len(ident)-1]
				}
			}
			if fname != "" {
				f := funcs[fname]
				if f != nil {
					count := 1
					if f.Context {
						count++
					}
					if f.Plural {
						count++
					}
					cmd := p.(*parse.CommandNode)
					// First argument is the function name
					if c := len(cmd.Args) - 1; c != count {
						log.Debugf("Skipping function %s (%v) - want %d arguments, got %d", f.Name, n.Position(), count, c)
						return
					}
					var s []string
					for ii := 1; ii < len(cmd.Args); ii++ {
						if sn, ok := cmd.Args[ii].(*parse.StringNode); ok {
							s = append(s, sn.Text)
						} else {
							log.Debugf("Skipping function %s (%v) - non-string argument at position %d", f.Name, n.Position(), ii)
							return
						}
					}
					message := &Message{}
					switch len(s) {
					case 1:
						message.Singular = s[0]
					case 2:
						if f.Context {
							message.Context = s[0]
							message.Singular = s[1]
						} else {
							message.Singular = s[0]
							message.Plural = s[1]
						}
					case 3:
						message.Context = s[0]
						message.Singular = s[1]
						message.Plural = s[2]
					}
					// TODO: The line number doesn't match exactly because of the
					// prepended variables
					pos := templatePosition(path, text, n)
					if err = messages.Add(message, pos, ""); err != nil {
						return
					}
				}
			}
		})
	}
	return err
}

func templatePosition(name string, text string, n parse.Node) *token.Position {
	return &token.Position{
		Filename: name,
		Line:     strings.Count(text[:int(n.Position())], "\n") + 1,
	}
}
