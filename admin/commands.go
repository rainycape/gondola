package admin

import (
	"fmt"
	"gnd.la/mux"
	"io"
	"os"
)

// Builtin admin commands implemented here
// rathen than in other packages to avoid
// import cycles.

func catFile(ctx *mux.Context) {
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

func init() {
	Register(catFile, &Options{
		Help:  "Prints a file from the blobstore to the stdout",
		Flags: Flags(BoolFlag("meta", false, "Print file metatada instead of file data")),
	})
}
