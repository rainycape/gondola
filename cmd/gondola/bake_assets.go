package main

import (
	"bytes"
	"fmt"
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/loaders"
	"gnd.la/log"
	"gnd.la/util/internal/gen/genutil"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

func BakeAssets(ctx *app.Context) {
	var dir string
	var name string
	extensions := []string{".html", ".css", ".js"}
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
	extensions = append(extensions, strings.Split(exts, ",")...)
	var out string
	ctx.ParseParamValue("o", &out)
	var compression string
	ctx.ParseParamValue("c", &compression)
	var compress loaders.BakeCompress
	switch compression {
	case "tgz":
		compress = loaders.CompressTgz
	case "zip":
		compress = loaders.CompressZip
	case "flate":
		compress = loaders.CompressFlate
	case "none":
		compress = loaders.CompressNone
	default:
		panic(fmt.Errorf("invalid compression method %q", compression))
	}
	var buf bytes.Buffer
	odir := filepath.Dir(out)
	p, err := build.ImportDir(odir, 0)
	if err == nil {
		buf.WriteString(fmt.Sprintf("package %s\n", p.Name))
	}
	buf.WriteString("import \"gnd.la/loaders\"\n")
	buf.WriteString(genutil.AutogenString())
	fmt.Fprintf(&buf, "var %s = ", name)
	if err := loaders.Bake(&buf, dir, extensions, compress); err != nil {
		panic(err)
	}
	if err := genutil.WriteAutogen(out, buf.Bytes()); err != nil {
		panic(err)
	}
	log.Debugf("Assets written to %s (%d bytes)", out, buf.Len())
}

func init() {
	admin.Register(BakeAssets, &admin.Options{
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
