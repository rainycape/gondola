package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gnd.la/app"
	"gnd.la/loaders"
	"gnd.la/log"
)

// Builtin admin commands implemented here
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
	a.TemplateDebug = false
	loader := a.TemplatesLoader()
	if names, err := loader.List(); err == nil {
		for _, name := range names {
			if _, err := a.LoadTemplate(name); err != nil {
				log.Errorf("error loading template %q: %s", name, err)
			}
		}
	} else {
		log.Errorf("error listing templates: %s", err)
	}
}

func printResources(ctx *app.Context) {
	var assets string
	var templates string
	if mgr := ctx.App().AssetsManager(); mgr != nil {
		if ldr, ok := mgr.Loader().(loaders.DirLoader); ok {
			assets = ldr.Dir()
		}
	}
	if ldr, ok := ctx.App().TemplatesLoader().(loaders.DirLoader); ok {
		templates = ldr.Dir()
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
	Register(catFile, &Options{
		Help:  "Prints a file from the blobstore to the stdout",
		Flags: Flags(BoolFlag("meta", false, "Print file metatada instead of file data")),
	})
	Register(makeAssets, &Options{
		Help: "Pre-compile and bundle all app assets",
	})
	Register(printResources, &Options{Name: "_print-resources"})
	Register(renderTemplate, &Options{
		Name:  "_render-template",
		Help:  "Render a template and print its output",
		Flags: Flags(StringFlag("o", "", "Output file. If empty or -, outputs to stdout")),
	})
}
