package messages

import (
	"bytes"
	"fmt"
	"gnd.la/gen"
	"gnd.la/i18n/po"
	"gnd.la/i18n/table"
	"go/build"
	"path/filepath"
)

func Compile(filename string, translations []*po.Po) error {
	var buf bytes.Buffer
	dir := filepath.Dir(filename)
	p, err := build.ImportDir(dir, 0)
	if err == nil {
		buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
	}
	buf.WriteString("import \"gnd.la/i18n/table\"\n")
	buf.WriteString(gen.AutogenString())
	buf.WriteString("func init() {\n")
	for _, v := range translations {
		table := poToTable(v)
		data, err := table.Encode()
		if err != nil {
			return err
		}
		buf.WriteString("table.Register(\"")
		buf.WriteString(v.Attrs["Language"])
		buf.WriteString("\", []byte{")
		for ii, b := range data {
			buf.WriteString(fmt.Sprintf("0x%02X", b))
			buf.WriteByte(',')
			if ii%8 == 0 && ii != len(data)-1 {
				buf.WriteByte('\n')
			}
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteString("})\n")
	}
	buf.WriteString("\n}\n")
	return gen.WriteAutogen(filename, buf.Bytes())
}

func poToTable(p *po.Po) *table.Table {
	translations := make(map[string]table.Translation)
	for _, v := range p.Messages {
		if empty(v.Translations) {
			continue
		}
		key := table.Key(v.Context, v.Singular, v.Plural)
		translations[key] = v.Translations
	}
	tbl, err := table.New(p.Attrs["Plural-Forms"], translations)
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
