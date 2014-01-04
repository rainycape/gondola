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
	"strings"
)

const appFilename = "app.yaml"

type Templates struct {
	Path  string            `yaml:"path"`
	Hooks map[string]string `yaml:"hooks"`
}

type App struct {
	Dir       string
	Name      string            `yaml:"name"`
	Handlers  map[string]string `yaml:"handlers"`
	Vars      map[string]string `yaml:"vars"`
	Templates *Templates        `yaml:"templates"`
	Assets    string            `yaml:"assets"`
}

func (app *App) Gen() error {
	pkg, err := genutil.NewPackage(app.Dir)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n\n", pkg.Name())
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("import (\n\"gnd.la/app\"\n\"gnd.la/loaders\"\n\"gnd.la/template\"\n\"gnd.la/template/assets\"\n)\n")
	buf.WriteString("var _ = loaders.FSLoader\n")
	buf.WriteString("var _ = template.New\n")
	buf.WriteString("var _ = assets.NewManager\n")
	fmt.Fprintf(&buf, "var (\n App *app.App\n)\n")
	buf.WriteString("func init() {\n")
	buf.WriteString("App = app.New()\n")
	fmt.Fprintf(&buf, "App.SetName(%q)\n", app.Name)
	if app.Assets != "" {
		fmt.Fprintf(&buf, "assetsLoader := loaders.FSLoader(%q)\n", filepath.Join(app.Dir, app.Assets))
		buf.WriteString("const prefix = \"/assets/\"\n")
		buf.WriteString("manager := assets.NewManager(assetsLoader, prefix)\n")
		buf.WriteString("App.SetAssetsManager(manager)\n")
		buf.WriteString("assetsHandler := assets.Handler(manager)\n")
		buf.WriteString("App.Handle(\"^\"+prefix, func(ctx *app.Context) { assetsHandler(ctx, ctx.R) })\n")
	}
	if app.Templates != nil && app.Templates.Path != "" {
		re := regexp.MustCompile("\\W")
		fmt.Fprintf(&buf, "templatesLoader := loaders.FSLoader(%q)\n", filepath.Join(app.Dir, app.Templates.Path))
		buf.WriteString("App.SetTemplatesLoader(templatesLoader)\n")
		for k, v := range app.Templates.Hooks {
			var pos string
			switch strings.ToLower(v) {
			case "top":
				pos = "assets.Top"
			case "bottom":
				pos = "assets.Bottom"
			case "none":
				pos = "assets.None"
			default:
				return fmt.Errorf("invalid hook position %q", v)
			}
			suffix := re.ReplaceAllString(k, "_")
			fmt.Fprintf(&buf, "tmpl_%s, err := App.LoadTemplate(%q)\n", suffix, k)
			buf.WriteString("if err != nil {\npanic(err)\n}\n")
			fmt.Fprintf(&buf, "App.AddHook(&template.Hook{Template: tmpl_%s.Template(), Position: %s})\n", suffix, pos)
		}
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
	if len(app.Vars) > 0 {
		buf.WriteString("App.AddTemplateVars(map[string]interface{}{\n")
		for k, v := range app.Vars {
			ident := k
			name := v
			if name == "" {
				name = ident
			}
			obj := scope.Lookup(ident)
			if obj == nil {
				return fmt.Errorf("could not find identifier named %q", ident)
			}
			rhs := ident
			if va, ok := obj.(*types.Var); ok {
				tn := va.Type().String()
				if strings.Contains(tn, ".") {
					tn = "interface{}"
				}
				rhs = fmt.Sprintf("func() %s { return %s }", tn, ident)
			}
			fmt.Fprintf(&buf, "%q: %s,\n", name, rhs)
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
