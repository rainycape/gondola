package admin

import (
	"fmt"
	"io"
	"os"

	"gnd.la/app"
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

func init() {
	Register(catFile, &Options{
		Help:  "Prints a file from the blobstore to the stdout",
		Flags: Flags(BoolFlag("meta", false, "Print file metatada instead of file data")),
	})
	Register(makeAssets, &Options{
		Help: "Pre-compile and bundle all app assets",
	})
}
