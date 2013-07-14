package messages

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"gondola/astutil"
	"gondola/log"
	"gondola/pkg"
	"os"
	"path/filepath"
	"strings"
)

func DefaultFunctions() []*Function {
	return []*Function{
		// Singular functions without context
		{Name: "gondola/i18n.T"},
		{Name: "gondola/i18n.Errorf"},
		{Name: "gondola/i18n.Sprintf"},
		{Name: "gondola/i18n.NewError"},
		{Name: "gondola/mux.Context.T"},
		{Name: "T", Template: true},
		// Singular functions with context
		{Name: "gondola/i18n.Tc", Context: true},
		{Name: "gondola/i18n.Sprintfc", Context: true},
		{Name: "gondola/i18n.Errorfc", Context: true},
		{Name: "gondola/i18n.NewErrorc", Context: true},
		{Name: "gondola/mux.Context.Tc", Context: true},
		{Name: "Tc", Template: true, Context: true},
		// Plural functions without context
		{Name: "gondola/i18n.Tn", Plural: true},
		{Name: "gondola/i18n.Sprintfn", Plural: true},
		{Name: "gondola/i18n.Errorfn", Plural: true},
		{Name: "gondola/i18n.NewErrorn", Plural: true},
		{Name: "gondola/mux.Context.Tn", Plural: true},
		{Name: "Tn", Template: true, Plural: true},
		// Plural functions with context
		{Name: "gondola/i18n.Tnc", Context: true, Plural: true},
		{Name: "gondola/i18n.Errorfnc", Context: true, Plural: true},
		{Name: "gondola/i18n.Sprintfnc", Context: true, Plural: true},
		{Name: "gondola/i18n.NewErrornc", Context: true, Plural: true},
		{Name: "gondola/mux.Context.Tnc", Context: true, Plural: true},
		{Name: "Tnc", Template: true, Context: true, Plural: true},
	}
}

func DefaultTypes() []string {
	return []string{
		"gondola/i18n.String",
	}
}

func DefaultTagFields() []string {
	return []string{
		"help",
		"label",
	}
}

func Extract(dir string, functions []*Function, types []string, tagFields []string) ([]*Message, error) {
	messages := make(messageMap)
	err := extract(messages, dir, functions, types, tagFields)
	if err != nil {
		return nil, err
	}
	return messages.Messages(), nil
}

func extract(messages messageMap, dir string, functions []*Function, types []string, tagFields []string) error {
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
			if !pkg.IsPackage(p) {
				if err := extract(messages, p, functions, types, tagFields); err != nil {
					return err
				}
			}
			continue
		}
		switch strings.ToLower(filepath.Ext(name)) {
		// TODO: templates, strings files
		case ".go":
			if err := extractGoMessages(messages, p, functions, types, tagFields); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractGoMessages(messages messageMap, path string, functions []*Function, types []string, tagFields []string) error {
	log.Debugf("Extracting messages from Go file %s", path)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing go file %s: %s", path, err)
	}
	for _, v := range functions {
		if v.Template {
			continue
		}
		if err := extractGoFunc(messages, fset, f, v); err != nil {
			return err
		}
	}
	for _, v := range types {
		if err := extractGoType(messages, fset, f, v); err != nil {
			return err
		}
	}
	for _, v := range tagFields {
		if err := extractGoTagField(messages, fset, f, v); err != nil {
			return err
		}
	}
	return nil
}

func extractGoFunc(messages messageMap, fset *token.FileSet, f *ast.File, fn *Function) error {
	calls, err := astutil.Calls(fset, f, fn.Name)
	if err != nil {
		return err
	}
	n := 0
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
				ctx, cpos := astutil.StringLiteral(fset, c.Args[0])
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
	tf := &Function{Name: typ}
	if err := extractGoFunc(messages, fset, f, tf); err != nil {
		return err
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
