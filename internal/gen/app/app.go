package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gnd.la/internal/gen/genutil"
	"gnd.la/internal/vfsutil"
	"gnd.la/log"
	"gnd.la/util/yaml"

	"code.google.com/p/go.tools/go/types"
)

const appFilename = "appfile.yaml"

type Templates struct {
	Path      string            `yaml:"path"`
	Functions map[string]string `yaml:"functions"`
	Hooks     map[string]string `yaml:"hooks"`
}

type Translations struct {
	Context string `yaml:"context"`
}

type App struct {
	Dir          string
	Name         string                 `yaml:"name"`
	Handlers     map[string]string      `yaml:"handlers"`
	Vars         map[string]interface{} `yaml:"vars"`
	Templates    *Templates             `yaml:"templates"`
	Translations *Translations          `yaml:"translations"`
	Assets       string                 `yaml:"assets"`
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
	if app.Assets != "" || (app.Templates != nil && app.Templates.Path != "" && len(app.Templates.Hooks) > 0) {
		buf.WriteString("var manager *assets.Manager\n")
	}
	if app.Assets != "" {
		buf.WriteString("assetsFS := ")
		if err := app.writeFS(&buf, filepath.Join(app.Dir, app.Assets), release); err != nil {
			return err
		}
		buf.WriteString("const prefix = \"/assets/\"\n")
		buf.WriteString("manager = assets.New(assetsFS, prefix)\n")
		buf.WriteString("App.SetAssetsManager(manager)\n")
		buf.WriteString("App.Handle(\"^\"+prefix, app.HandlerFromHTTPFunc(manager.Handler()))\n")
	}
	scope := pkg.Scope()
	if len(app.Vars) > 0 {
		var varNames []string
		for k := range app.Vars {
			varNames = append(varNames, k)
		}
		sort.Strings(varNames)
		buf.WriteString("App.AddTemplateVars(map[string]interface{}{\n")
		for _, k := range varNames {
			v := app.Vars[k]
			ident := k
			name := v
			if name == "" || name == nil {
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
	var handlerNames []string
	for k := range app.Handlers {
		handlerNames = append(handlerNames, k)
	}
	sort.Strings(handlerNames)
	for _, k := range handlerNames {
		v := app.Handlers[k]
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
	if app.Templates != nil {
		if len(app.Templates.Functions) > 0 {
			buf.WriteString("template.AddFuncs(template.FuncMap{\n")
			for k, v := range app.Templates.Functions {
				obj := scope.Lookup(v)
				if obj == nil {
					return fmt.Errorf("could not find function named %q for template function %q", v, k)
				}
				fmt.Fprintf(&buf, "%q: %s,\n", k, v)
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
				fmt.Fprintf(&buf, "%s := template.New(templatesFS, manager)\n", name)
				fmt.Fprintf(&buf, "%s.Funcs(map[string]interface{}{\n", name)
				funcNames := []string{"t", "tn", "tc", "tnc", "reverse"}
				for _, v := range funcNames {
					fmt.Fprintf(&buf, "\"%s\": func(_ ...interface{}) interface{} { return nil },\n", v)
				}
				buf.WriteString("})\n")
				fmt.Fprintf(&buf, "if err := %s.Parse(%q); err != nil {\npanic(err)\n}\n", name, k)
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
	var app *App
	if err := yaml.Unmarshal(data, &app); err != nil {
		return nil, err
	}
	app.Dir = dir
	return app, nil
}
