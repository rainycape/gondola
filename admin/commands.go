package admin

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/loaders"
	"gnd.la/log"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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

func makeAppAssets(a *app.App) {
	a.SetDebug(false)
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

func makeAssets(ctx *app.Context) {
	makeAppAssets(ctx.App())
}

func rmGen(ctx *app.Context) {
	if am := ctx.App().AssetsManager(); am != nil {
		if dl, ok := am.Loader().(loaders.DirLoader); ok {
			re := regexp.MustCompile("(?i).+\\.gen\\..+")
			filepath.Walk(dl.Dir(), func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() && re.MatchString(path) {
					log.Debugf("Removing %s", path)
					if err := os.Remove(path); err != nil {
						panic(err)
					}
					dir := filepath.Dir(path)
					if infos, err := ioutil.ReadDir(dir); err == nil && len(infos) == 0 {
						log.Debugf("Removing empty dir %s", dir)
						if err := os.Remove(dir); err != nil {
							panic(err)
						}
					}
				}
				return nil
			})
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
	Register(rmGen, &Options{
		Help: "Remove Gondola generated files (identifier by *.gen.*)",
	})
}
