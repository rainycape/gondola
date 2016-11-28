package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gnd.la/app"
	"gnd.la/log"

	"github.com/rainycape/vfs"
)

// Builtin commands implemented here
// rathen than in other packages to avoid
// import cycles.

func catFile(ctx *app.Context) {
	var id string
	ctx.MustParseIndexValue(0, &id)
	f, err := ctx.Blobstore().Open(id)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var meta bool
	ctx.ParseParamValue("meta", &meta)
	if meta {
		var m interface{}
		if err := f.GetMeta(&m); err != nil {
			panic(err)
		}
		fmt.Println(m)
	} else {
		io.Copy(os.Stdout, f)
	}
}

func makeAssets(ctx *app.Context) {
	a := ctx.App()
	if cfg := a.Config(); cfg != nil {
		cfg.TemplateDebug = false
	}
	err := vfs.Walk(a.TemplatesFS(), "/", func(fs vfs.VFS, p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || p == "" || p[0] == '.' {
			return err
		}
		if _, err := a.LoadTemplate(p); err != nil {
			log.Errorf("error loading template %s: %s", p, err)
		}
		return nil
	})

	if err != nil {
		log.Errorf("error listing templates: %s", err)
	}
}

func printResources(ctx *app.Context) {
	// TODO: Define an interface in package vfs, so this fails
	// if the interface is changed or renamed.
	type rooter interface {
		Root() string
	}
	var assets string
	var templates string
	if mgr := ctx.App().AssetsManager(); mgr != nil {
		if r, ok := mgr.VFS().(rooter); ok {
			assets = r.Root()
		}
	}
	if r, ok := ctx.App().TemplatesFS().(rooter); ok {
		templates = r.Root()
	}
	resources := map[string]string{
		"assets":    assets,
		"templates": templates,
	}
	data, err := json.Marshal(resources)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}

func renderTemplate(ctx *app.Context) {
	var template string
	ctx.MustParseIndexValue(0, &template)
	tmpl, err := ctx.App().LoadTemplate(template)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTo(&buf, ctx, nil); err != nil {
		panic(err)
	}
	var output string
	ctx.ParseParamValue("o", &output)
	if output == "" || output == "-" {
		fmt.Print(buf.String())
	} else {
		if err := ioutil.WriteFile(output, buf.Bytes(), 0644); err != nil {
			panic(err)
		}
	}
}

func init() {
	MustRegister(catFile,
		Help("Prints a file from the blobstore to the stdout"),
		BoolFlag("meta", false, "Print file metatada instead of file data"),
	)
	MustRegister(makeAssets, Help("Pre-compile and bundle all app assets"))
	MustRegister(printResources, Name("_print-resources"))
	MustRegister(renderTemplate,
		Name("_render-template"),
		Help("Render a template and print its output"),
		StringFlag("o", "", "Output file. If empty or -, outputs to stdout"),
	)
}
