package app

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"gnd.la/gen/genutil"
	"gnd.la/log"
	"io/ioutil"
	"launchpad.net/goyaml"
	"path/filepath"
	"regexp"
)

const appFilename = "app.yaml"

type App struct {
	Dir       string
	Name      string            `yaml:"name"`
	Handlers  map[string]string `yaml:"handlers"`
	Exports   map[string]string `yaml:"exports"`
	Templates string            `yaml:"templates"`
	Assets    string            `yaml:"assets"`
}

func (app *App) Gen() error {
	pkg, err := genutil.NewPackage(app.Dir)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n\n", pkg.Name())
	buf.WriteString("import (\n\"gnd.la/app\"\n\"gnd.la/loaders\"\n\"gnd.la/template/assets\"\n)\n")
	buf.WriteString("var _ = loaders.FSLoader\n")
	buf.WriteString("var _ = assets.NewManager\n")
	buf.WriteString(genutil.AutogenString())
	fmt.Fprintf(&buf, "var (\n App *app.App\n)\n")
	buf.WriteString("func init() {\n")
	buf.WriteString("App = app.New()\n")
	fmt.Fprintf(&buf, "App.Name = %q\n", app.Name)
	if app.Templates != "" {
		fmt.Fprintf(&buf, "templatesLoader := loaders.FSLoader(%q)\n", filepath.Join(app.Dir, app.Templates))
		buf.WriteString("App.SetTemplatesLoader(templatesLoader)\n")
	}
	if app.Assets != "" {
		fmt.Fprintf(&buf, "assetsLoader := loaders.FSLoader(%q)\n", filepath.Join(app.Dir, app.Assets))
		buf.WriteString("const prefix = \"assets\"\n")
		buf.WriteString("manager := assets.NewManager(assetsLoader, prefix)\n")
		buf.WriteString("App.SetAssetsManager(manager)\n")
		buf.WriteString("assetsHandler := assets.Handler(manager)\n")
		buf.WriteString("App.Handle(\"^\"+prefix, func(ctx *app.Context) { assetsHandler(ctx, ctx.R) })\n")
	}
	scope := pkg.Scope()
	for k, v := range app.Handlers {
		obj := scope.Lookup(k)
		if obj == nil {
			return fmt.Errorf("could not find handler named %q", k)
		}
		if _, err := regexp.Compile(v); err != nil {
			return fmt.Errorf("invalid pattern %q: %s", v, err)
		}
		switch obj.Type().String() {
		case "*gnd.la/app.HandlerInfo", "gnd.la/app.HandlerInfo":
			fmt.Fprintf(&buf, "App.HandleOptions(%q, %s.Handler, %s.Options)\n", v, obj.Name(), obj.Name())
		default:
			return fmt.Errorf("invalid handler type %s", obj.Type())
		}
	}
	if len(app.Exports) > 0 {
		buf.WriteString("App.Export(map[string]interface{}{\n")
		for k, v := range app.Exports {
			varname := k
			key := v
			if key == "" {
				key = k
			}
			obj := scope.Lookup(k)
			if obj == nil {
				return fmt.Errorf("could not find export named %q", k)
			}
			if _, ok := obj.(*types.Var); ok {
				varname = "&" + varname
			}
			fmt.Fprintf(&buf, "%q: %s,\n", key, varname)
		}
		buf.WriteString("})\n")
	}
	buf.WriteString("}\n")
	out := filepath.Join(pkg.Dir(), "gondola_app.go")
	log.Debugf("Writing Gondola app to %s", out)
	return genutil.WriteAutogen(out, buf.Bytes())
}

func Parse(dir string) (*App, error) {
	appFile := filepath.Join(dir, appFilename)
	data, err := ioutil.ReadFile(appFile)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %s", appFilename, err)
	}
	var app *App
	if err := goyaml.Unmarshal(data, &app); err != nil {
		return nil, err
	}
	app.Dir = dir
	return app, nil
}
