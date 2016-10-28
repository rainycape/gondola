package app

import (
	"bytes"
	"fmt"
	"go/types"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"gnd.la/internal/gen/genutil"
	"gnd.la/internal/vfsutil"
	"gnd.la/log"

	"github.com/naoina/toml"
)

const appFilename = "appfile.toml"

type Assets struct {
	Path string `toml:"path"`
}

type Var struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

type Hook struct {
	Name     string `toml:"name"`
	Position string `toml:"position"`
}

type TemplateFunc struct {
	Name   string `toml:"name"`
	GoFunc string `toml:"fn"`
}

type Templates struct {
	Path      string         `toml:"path"`
	Functions []TemplateFunc `toml:"functions"`
	Hooks     []Hook         `toml:"hooks"`
	Vars      []Var          `toml:"vars"`
}

type Translations struct {
	Context string `toml:"context"`
}

type Handler struct {
	Path    string `toml:"path"`
	Handler string `toml:"handler"`
	Name    string `toml:"name"`
}

type App struct {
	Dir          string
	Name         string        `toml:"name"`
	Handlers     []Handler     `toml:"handlers"`
	Templates    *Templates    `toml:"templates"`
	Translations *Translations `toml:"translations"`
	Assets       *Assets       `toml:"assets"`
}

func (app *App) writeFS(buf *bytes.Buffer, dir string, release bool) error {
	if release {
		return vfsutil.BakedFS(buf, dir, nil)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "vfsutil.MemFromDir(%q)\n", abs)
	return nil
}

func (app *App) assetsPath() string {
	if app.Assets != nil {
		return app.Assets.Path
	}
	return ""
}

func (app *App) Gen(release bool) error {
	pkg, err := genutil.NewPackage(app.Dir)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n\n", pkg.Name())
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("import (\n\"gnd.la/app\"\n\"gnd.la/internal/vfsutil\"\n\"gnd.la/template\"\n\"gnd.la/template/assets\"\n)\n")
	buf.WriteString("var _ = vfsutil.Bake\n")
	buf.WriteString("var _ = template.New\n")
	buf.WriteString("var _ = assets.New\n")
	fmt.Fprintf(&buf, "var (\n App = app.New()\n)\n")
	buf.WriteString("func init() {\n")
	// TODO: Enable this when we have a solution for
	// executing templates from differnt apps from the
	// same *app.Context, which would need different
	// default translation contexts.
	//
	/*if app.Translations != nil && app.Translations.Context != "" {
		fmt.Fprintf(&buf, "App.TranslationContext = %q\n", app.Translations.Context)
	}*/
	fmt.Fprintf(&buf, "App.SetName(%q)\n", app.Name)
	if app.assetsPath() != "" || (app.Templates != nil && app.Templates.Path != "" && len(app.Templates.Hooks) > 0) {
		buf.WriteString("var manager *assets.Manager\n")
	}
	if app.assetsPath() != "" {
		buf.WriteString("assetsFS := ")
		if err := app.writeFS(&buf, filepath.Join(app.Dir, app.assetsPath()), release); err != nil {
			return err
		}
		buf.WriteString("const prefix = \"/assets/\"\n")
		buf.WriteString("manager = assets.New(assetsFS, prefix)\n")
		buf.WriteString("App.SetAssetsManager(manager)\n")
		buf.WriteString("App.Handle(\"^\"+prefix, app.HandlerFromHTTPFunc(manager.Handler()))\n")
	}
	scope := pkg.Scope()
	if app.Templates != nil && len(app.Templates.Vars) > 0 {
		buf.WriteString("App.AddTemplateVars(map[string]interface{}{\n")
		for _, v := range app.Templates.Vars {
			if v.Name == "" {
				return fmt.Errorf("missing var name in %v", v)
			}
			value := v.Value
			if value == "" {
				value = v.Name
			}
			obj := scope.Lookup(value)
			if obj == nil {
				return fmt.Errorf("could not find identifier named %q", value)
			}
			rhs := value
			if va, ok := obj.(*types.Var); ok {
				tn := va.Type().String()
				if strings.Contains(tn, ".") {
					tn = "interface{}"
				}
				rhs = fmt.Sprintf("func() %s { return %s }", tn, value)
			}
			fmt.Fprintf(&buf, "%q: %s,\n", v.Name, rhs)
		}
		buf.WriteString("})\n")
	}
	for _, v := range app.Handlers {
		handlerFunc := v.Handler
		if handlerFunc == "" {
			panic(fmt.Errorf("missing handler in %+v", v))
		}
		obj := scope.Lookup(handlerFunc)
		if obj == nil {
			return fmt.Errorf("could not find handler named %q", handlerFunc)
		}
		if _, err := regexp.Compile(v.Path); err != nil {
			return fmt.Errorf("invalid pattern %q: %s", v.Path, err)
		}
		handlerName := v.Name
		if handlerName == "" {
			handlerName = handlerFunc + "Name"
		}
		fmt.Fprintf(&buf, "App.Handle(%q, %s, app.NamedHandler(%s))\n", v.Path, handlerFunc, handlerName)
	}
	if app.Templates != nil {
		if len(app.Templates.Functions) > 0 {
			buf.WriteString("template.AddFuncs(template.FuncMap{\n")
			for _, v := range app.Templates.Functions {
				obj := scope.Lookup(v.GoFunc)
				if obj == nil {
					return fmt.Errorf("could not find function named %q for template function %q", v.GoFunc, v.Name)
				}
				fmt.Fprintf(&buf, "%q: %s,\n", v.Name, v.GoFunc)
			}
			buf.WriteString("})\n")
		}
		if app.Templates.Path != "" {
			buf.WriteString("templatesFS := ")
			if err := app.writeFS(&buf, filepath.Join(app.Dir, app.Templates.Path), release); err != nil {
				return err
			}
			buf.WriteString("App.SetTemplatesFS(templatesFS)\n")
			re := regexp.MustCompile("\\W")
			for _, v := range app.Templates.Hooks {
				var pos string
				switch strings.ToLower(v.Position) {
				case "top":
					pos = "assets.Top"
				case "bottom":
					pos = "assets.Bottom"
				case "none":
					pos = "assets.None"
				default:
					return fmt.Errorf("invalid hook position %q", v.Position)
				}
				suffix := re.ReplaceAllString(v.Name, "_")
				name := fmt.Sprintf("tmpl_%s", suffix)
				fmt.Fprintf(&buf, "%s := template.New(templatesFS, manager)\n", name)
				fmt.Fprintf(&buf, "%s.Funcs(map[string]interface{}{\n", name)
				funcNames := []string{"t", "tn", "tc", "tnc", "reverse"}
				for _, v := range funcNames {
					fmt.Fprintf(&buf, "\"%s\": func(_ ...interface{}) interface{} { return nil },\n", v)
				}
				buf.WriteString("})\n")
				fmt.Fprintf(&buf, "if err := %s.Parse(%q); err != nil {\npanic(err)\n}\n", name, v.Name)
				fmt.Fprintf(&buf, "App.AddHook(&template.Hook{Template: %s, Position: %s})\n", name, pos)
			}
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
	var app App
	if err := toml.Unmarshal(data, &app); err != nil {
		return nil, err
	}
	app.Dir = dir
	return &app, nil
}
