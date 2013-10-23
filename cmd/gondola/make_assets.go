package main

import (
	"bytes"
	"compress/flate"
	"fmt"
	"gnd.la/admin"
	"gnd.la/mux"
	"gnd.la/util"
	"go/build"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func MakeAssets(ctx *mux.Context) {
	var dir string
	var name string
	extensions := map[string]struct{}{
		".html": struct{}{},
		".css":  struct{}{},
		".js":   struct{}{},
	}
	ctx.ParseParamValue("dir", &dir)
	if dir == "" {
		fmt.Fprintf(os.Stderr, "dir can't be empty\n")
		return
	}
	ctx.ParseParamValue("name", &name)
	if name == "" {
		fmt.Fprintf(os.Stderr, "name can't be empty\n")
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
	ctx.ParseParamValue("o", &out)
	var useFlate bool
	ctx.ParseParamValue("flate", &useFlate)
	var buf bytes.Buffer
	if out != "" {
		// Try to guess package name. Do it before writing the file, otherwise the package becomes invalid.
		odir := filepath.Dir(out)
		p, err := build.ImportDir(odir, 0)
		if err == nil {
			buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
		}
	}
	buf.WriteString("import \"gnd.la/loaders\"\n")
	buf.WriteString(autogenString())
	if useFlate {
		buf.WriteString(fmt.Sprintf("var %s = loaders.FlateLoader(loaders.MapLoader(map[string][]byte{\n", name))
	} else {
		buf.WriteString(fmt.Sprintf("var %s = loaders.MapLoader(map[string][]byte{\n", name))
	}
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
				if useFlate {
					var cbuf bytes.Buffer
					w, err := flate.NewWriter(&cbuf, flate.BestCompression)
					if err != nil {
						return fmt.Errorf("error compressing %s: %s", path, err)
					}
					if _, err := w.Write(contents); err != nil {
						return fmt.Errorf("error compressing %s: %s", path, err)
					}
					if err := w.Close(); err != nil {
						return fmt.Errorf("error compressing %s: %s", path, err)
					}
					contents = cbuf.Bytes()
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
	buf.WriteString("})")
	if useFlate {
		buf.WriteString(")")
	}
	buf.WriteString("\n")
	b, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	var force bool
	ctx.ParseParamValue("f", &force)
	force = force || isAutogen(out)
	if err := util.WriteFile(out, b, force, 0644); err != nil {
		panic(err)
	}
}

func init() {
	admin.Register(MakeAssets, &admin.Options{
		Help: "Converts all assets in <dir> into Go code and generates a Loader named with <name>",
		Flags: admin.Flags(
			admin.StringFlag("dir", "", "Directory with the html templates"),
			admin.StringFlag("name", "", "Name of the generated MapLoader"),
			admin.StringFlag("o", "", "Output filename. If empty, output is printed to standard output"),
			admin.BoolFlag("flate", false, "Compress resources with flate when generating the code"),
			admin.BoolFlag("f", false, "When creating the output file, overwrite any existing file with the same name"),
			admin.StringFlag("extensions", "", "Additional extensions (besides html, css and js) to include, separated by commas"),
		),
	})
}
