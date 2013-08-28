package main

import (
	"bytes"
	"fmt"
	"go/build"
	"go/format"
	"gondola/admin"
	"gondola/mux"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func TemplatesMap(ctx *mux.Context) {
	var dir string
	var name string
	extensions := map[string]struct{}{
		".html": struct{}{},
	}
	ctx.ParseParamValue("dir", &dir)
	if dir == "" {
		fmt.Fprintf(os.Stderr, "dir can't be empty")
		return
	}
	ctx.ParseParamValue("name", &name)
	if name == "" {
		fmt.Fprintf(os.Stderr, "name can't be empty")
		return
	}
	var exts string
	ctx.ParseParamValue("extensions", &exts)
	if exts != "" {
		for _, v := range strings.Split(exts, ",") {
			e := strings.ToLower(strings.TrimSpace(v))
			if e != "" {
				if e[0] != '.' {
					e = "." + e
				}
				extensions[e] = struct{}{}
			}
		}
	}
	var out string
	ctx.ParseParamValue("out", &out)
	var buf bytes.Buffer
	if out != "" {
		// Try to guess package name. Do it before writing the file, otherwise the package becomes invalid.
		odir := filepath.Dir(out)
		p, err := build.ImportDir(odir, 0)
		if err == nil {
			buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
		}
	}
	buf.WriteString("import \"gondola/loaders\"\n")
	buf.WriteString(fmt.Sprintf("// AUTOMATICALLY GENERATED WITH %s. DO NOT EDIT!\n", strings.Join(os.Args, " ")))
	buf.WriteString(fmt.Sprintf("var %s = loaders.MapLoader(map[string][]byte{\n", name))
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			if _, ok := extensions[strings.ToLower(filepath.Ext(path))]; ok {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading %s: %s", path, err)
				}
				rel := path[len(dir):]
				if rel[0] == '/' {
					rel = rel[1:]
				}
				buf.WriteString(fmt.Sprintf("%q", rel))
				buf.WriteByte(':')
				buf.WriteString(" []byte{")
				for ii, v := range contents {
					buf.WriteString(fmt.Sprintf("0x%02X", v))
					buf.WriteByte(',')
					if ii%8 == 0 {
						buf.WriteByte('\n')
					}
				}
				buf.Truncate(buf.Len() - 1)
				buf.WriteString("},\n")
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	buf.WriteString("})\n")
	b, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	var w io.Writer
	if out != "" {
		flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		var force bool
		ctx.ParseParamValue("f", &force)
		if !force {
			flags |= os.O_EXCL
		}
		f, err := os.OpenFile(out, flags, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating output file %q: %s\n", out, err)
			return
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}
	_, err = w.Write(b)
	if err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(TemplatesMap, &admin.Options{
		Help: "Converts all templates in <dir> into Go code and generates a MapLoader named with <name>",
		Flags: admin.Flags(
			admin.StringFlag("dir", "", "Directory with the html templates"),
			admin.StringFlag("name", "", "Name of the generated MapLoader"),
			admin.StringFlag("out", "", "Output filename. If empty, output is printed to standard output"),
			admin.BoolFlag("f", false, "When creating output file, overwrite any existing file with the same name"),
			admin.StringFlag("extensions", "", "Additional extensions (besides .html) to include, separated by commas"),
		),
	})
}
