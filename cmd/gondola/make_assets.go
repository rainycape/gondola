package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/gen/genutil"
	"gnd.la/log"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func writeBytes(buf *bytes.Buffer, data []byte, raw bool) {
	if raw {
		buf.WriteString(fmt.Sprintf("loaders.RawString(%q)", string(data)))
	} else {
		buf.WriteString("[]byte{")
		for ii, v := range data {
			s := fmt.Sprintf("%q,", v)
			if len(s) > 3 {
				if s[2] == 'u' {
					s = "'\\x" + s[5:]
				} else if s == "'\\x00'," {
					s = "0,"
				}
			}
			buf.WriteString(s)
			if ii%16 == 0 && ii != len(data)-1 {
				buf.WriteByte('\n')
			}
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteString("}")
	}
}

func flateCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func zipCompress(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for k, v := range files {
		f, err := w.Create(k)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(v); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func tgzCompress(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	z, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	w := tar.NewWriter(z)
	for k, v := range files {
		hdr := &tar.Header{
			Name: k,
			Size: int64(len(v)),
		}
		if err := w.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := w.Write(v); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func MakeAssets(ctx *app.Context) {
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
	files := make(map[string][]byte)
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
				files[rel] = contents
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	var out string
	ctx.ParseParamValue("o", &out)
	var compression string
	ctx.ParseParamValue("c", &compression)
	var buf bytes.Buffer
	odir := filepath.Dir(out)
	p, err := build.ImportDir(odir, 0)
	if err == nil {
		buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
	}
	buf.WriteString("import \"gnd.la/loaders\"\n")
	buf.WriteString(genutil.AutogenString())
	switch compression {
	case "tgz":
		buf.WriteString(fmt.Sprintf("var %s = loaders.TgzLoader(", name))
		data, err := tgzCompress(files)
		if err != nil {
			panic(fmt.Errorf("error compressing tgz: %s", err))
		}
		log.Debugf("Compressed with tgz to %d bytes", len(data))
		writeBytes(&buf, data, true)
		buf.WriteString(")\n")
	case "zip":
		buf.WriteString(fmt.Sprintf("var %s = loaders.ZipLoader(", name))
		data, err := zipCompress(files)
		if err != nil {
			panic(fmt.Errorf("error compressing zip: %s", err))
		}
		log.Debugf("Compressed with zip to %d bytes", len(data))
		writeBytes(&buf, data, true)
		buf.WriteString(")\n")
	case "flate":
		buf.WriteString(fmt.Sprintf("var %s = loaders.FlateLoader(loaders.MapLoader(map[string][]byte{\n", name))
		for k, v := range files {
			buf.WriteString(fmt.Sprintf("%q", k))
			buf.WriteString(": ")
			cmp, err := flateCompress(v)
			if err != nil {
				panic(fmt.Errorf("error compressing %s with flate: %s", k, err))
			}
			writeBytes(&buf, cmp, false)
			buf.WriteString(",\n")
		}
		buf.WriteString("}))\n")
	case "none":
		buf.WriteString(fmt.Sprintf("var %s = loaders.MapLoader(map[string][]byte{\n", name))
		for k, v := range files {
			buf.WriteString(fmt.Sprintf("%q", k))
			buf.WriteString(": ")
			writeBytes(&buf, v, false)
			buf.WriteString(",\n")
		}
		buf.WriteString("})\n")
	default:
		panic(fmt.Errorf("invalid compression method %q", compression))
	}
	if err := genutil.WriteAutogen(out, buf.Bytes()); err != nil {
		panic(err)
	}
	log.Debugf("Assets written to %s (%d bytes)", out, buf.Len())
}

func init() {
	admin.Register(MakeAssets, &admin.Options{
		Help: "Converts all assets in <dir> into Go code and generates a Loader named with <name>",
		Flags: admin.Flags(
			admin.StringFlag("dir", "", "Directory with the html templates"),
			admin.StringFlag("name", "", "Name of the generated MapLoader"),
			admin.StringFlag("o", "", "Output filename. If empty, output is printed to standard output"),
			admin.StringFlag("c", "tgz", "Compress type to use. tgz|zip|flate|none"),
			admin.StringFlag("extensions", "", "Additional extensions (besides html, css and js) to include, separated by commas"),
		),
	})
}
