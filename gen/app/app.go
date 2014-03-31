package app

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"gnd.la/gen/genutil"
	"gnd.la/loaders"
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

type Translations struct {
	Context string `yaml:"context"`
}

type App struct {
	Dir          string
	Name         string            `yaml:"name"`
	Handlers     map[string]string `yaml:"handlers"`
	Vars         map[string]string `yaml:"vars"`
	Templates    *Templates        `yaml:"templates"`
	Translations *Translations     `yaml:"translations"`
	Assets       string            `yaml:"assets"`
}

func (app *App) writeLoader(buf *bytes.Buffer, dir string, release bool) error {
	if release {
		return loaders.Bake(buf, dir, nil, loaders.CompressTgz)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "loaders.MemLoader(loaders.FSLoader(%q))\n", abs)
	return nil
}

func (app *App) Gen(release bool) error {
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
	// TODO: Enable this when we have a solution for
	// executing templates from differnt apps from the
	// same *app.Context, which would need different
	// default translation contexts.
	//
	/*if app.Translations != nil && app.Translations.Context != "" {
		fmt.Fprintf(&buf, "App.TranslationContext = %q\n", app.Translations.Context)
	}*/
	fmt.Fprintf(&buf, "App.SetName(%q)\n", app.Name)
	buf.WriteString("var manager *assets.Manager\n")
	if app.Assets != "" {
		buf.WriteString("assetsLoader := ")
		if err := app.writeLoader(&buf, filepath.Join(app.Dir, app.Assets), release); err != nil {
			return err
		}
		buf.WriteString("const prefix = \"/assets/\"\n")
		buf.WriteString("manager = assets.NewManager(assetsLoader, prefix)\n")
		buf.WriteString("App.SetAssetsManager(manager)\n")
		buf.WriteString("assetsHandler := assets.Handler(manager)\n")
		buf.WriteString("App.Handle(\"^\"+prefix, func(ctx *app.Context) { assetsHandler(ctx, ctx.R) })\n")
	}
	scope := pkg.Scope()
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
	if app.Templates != nil && app.Templates.Path != "" {
		buf.WriteString("templatesLoader := ")
		if err := app.writeLoader(&buf, filepath.Join(app.Dir, app.Templates.Path), release); err != nil {
			return err
		}
		buf.WriteString("App.SetTemplatesLoader(templatesLoader)\n")
		re := regexp.MustCompile("\\W")
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
			name := fmt.Sprintf("tmpl_%s", suffix)
			fmt.Fprintf(&buf, "%s := template.New(templatesLoader, manager)\n", name)
			fmt.Fprintf(&buf, "if err := %s.Parse(%q); err != nil {\npanic(err)\n}\n", name, k)
			fmt.Fprintf(&buf, "App.AddHook(&template.Hook{Template: %s, Position: %s})\n", name, pos)
		}
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
