package messages

import (
	"bytes"
	"fmt"
	"gnd.la/i18n/po"
	"gnd.la/i18n/table"
	"gnd.la/util"
	"go/build"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

func Compile(filename string, overwrite bool, translations []*po.Po) error {
	var buf bytes.Buffer
	dir := filepath.Dir(filename)
	p, err := build.ImportDir(dir, 0)
	if err == nil {
		buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
	}
	buf.WriteString("import \"gnd.la/i18n/table\"\n")
	buf.WriteString(fmt.Sprintf("// AUTOMATICALLY GENERATED WITH %s -- DO NOT EDIT!\n", strings.Join(os.Args, " ")))
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
	b, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := util.WriteFile(filename, b, overwrite, 0644); err != nil {
		return err
	}
	return nil
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
