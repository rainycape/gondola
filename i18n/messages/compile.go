package messages

import (
	"bytes"
	"fmt"
	"gnd.la/i18n/po"
	"gnd.la/i18n/table"
	"gnd.la/internal/gen/genutil"
	"go/build"
	"path/filepath"
)

type CompileOptions struct {
	DefaultContext string
}

func Compile(filename string, translations []*po.Po, opts *CompileOptions) error {
	var buf bytes.Buffer
	dir := filepath.Dir(filename)
	p, err := build.ImportDir(dir, 0)
	if err == nil {
		fmt.Fprintf(&buf, "package %s\n", p.Name)
	}
	buf.WriteString("import \"gnd.la/i18n/table\"\n")
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("func init() {\n")
	var defaultContext string
	if opts != nil {
		defaultContext = opts.DefaultContext
	}
	for _, v := range translations {
		table := poToTable(v, defaultContext)
		form, err := funcFromFormula(v.Attrs["Plural-Forms"])
		if err != nil {
			return err
		}
		data, err := table.Encode()
		if err != nil {
			return err
		}
		fmt.Fprintf(&buf, "table.Register(%q, func (n int) int {\n%s\n}, %q)\n", v.Attrs["Language"], form, data)
	}
	buf.WriteString("\n}\n")
	return genutil.WriteAutogen(filename, buf.Bytes())
}

func poToTable(p *po.Po, ctx string) *table.Table {
	translations := make(map[string]table.Translation)
	for _, v := range p.Messages {
		if v.Context == "" {
			v.Context = ctx
		}
		if empty(v.Translations) {
			continue
		}
		key := table.Key(v.Context, v.Singular, v.Plural)
		translations[key] = v.Translations
	}
	tbl, err := table.New(nil, translations)
	// This shouldn't happen because the formula was validated when loading
	// the .po file.
	if err != nil {
		panic(err)
	}
	return tbl
}

func empty(s []string) bool {
	for _, v := range s {
		if v != "" {
			return false
		}
	}
	return true
}
