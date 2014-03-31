package app

import (
	"fmt"
	"gnd.la/app/profile"
	"gnd.la/loaders"
	"gnd.la/template"
	"gnd.la/template/assets"
	"gnd.la/util/internal/templateutil"
	"io"
	"net/http"
	"os"
	"text/template/parse"
)

var (
	reservedVariables     = []string{"Ctx", "Request"}
	internalAssetsManager = assets.NewManager(appAssets, assetsPrefix)
	profileHook           *template.Hook
)

type Template interface {
	Template() *template.Template
	Execute(w io.Writer, data interface{}) error
	ExecuteTemplate(w io.Writer, name string, data interface{}) error
	ExecuteVars(w io.Writer, data interface{}, vars map[string]interface{}) error
	ExecuteTemplateVars(w io.Writer, name string, data interface{}, vars map[string]interface{}) error
}

type TemplateProcessor func(*template.Template) (*template.Template, error)

type tmpl struct {
	tmpl *template.Template
	app  *App
}

func (t *tmpl) reverse(name string, args ...interface{}) (string, error) {
	if t.app != nil {
		return t.app.reverse(name, args)
	}
	return "", fmt.Errorf("can't reverse %s because the app is not available", name)
}

func newTemplate(app *App, loader loaders.Loader, manager *assets.Manager) *tmpl {
	t := &tmpl{}
	t.app = app
	t.tmpl = template.New(loader, manager)
	t.tmpl.Debug = app.TemplateDebug
	t.tmpl.Funcs(template.FuncMap{
		"#reverse": t.reverse,
		"t":        nop,
		"tn":       nop,
		"tc":       nop,
		"tnc":      nop,
		templateutil.BeginTranslatableBlock: nop,
		templateutil.EndTranslatableBlock:   nop,
	})
	return t
}

func newAppTemplate(app *App) *tmpl {
	return newTemplate(app, app.templatesLoader, app.assetsManager)
}

func newInternalTemplate(app *App) *tmpl {
	return newTemplate(app, appAssets, internalAssetsManager)
}

func (t *tmpl) ParseVars(file string, vars template.VarMap) error {
	if vars != nil {
		for _, k := range reservedVariables {
			vars[k] = nil
		}
	}
	return t.tmpl.ParseVars(file, vars)
}

func (t *tmpl) Parse(file string) error {
	return t.ParseVars(file, nil)
}

func (t *tmpl) execute(w io.Writer, name string, data interface{}, vars template.VarMap) error {
	ctx, _ := w.(*Context)
	return t.executeContext(ctx, w, name, data, vars)
}

func (t *tmpl) executeContext(ctx *Context, w io.Writer, name string, data interface{}, vars template.VarMap) error {
	// TODO: Don't ignore received vars
	var request *http.Request
	if ctx != nil {
		request = ctx.R
	}
	var tvars map[string]interface{}
	var err error
	if t.app.namespace != nil {
		tvars, err = t.app.namespace.eval(ctx)
		if err != nil {
			return err
		}
	} else {
		tvars = make(map[string]interface{})
	}
	tvars["Ctx"] = ctx
	tvars["Request"] = request
	if name == "" {
		name = t.tmpl.Root()
	}
	return t.tmpl.ExecuteTemplateVars(w, name, data, tvars)
}

func (t *tmpl) Template() *template.Template {
	return t.tmpl
}

func (t *tmpl) Execute(w io.Writer, data interface{}) error {
	return t.execute(w, "", data, nil)
}

func (t *tmpl) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return t.execute(w, name, data, nil)
}

func (t *tmpl) ExecuteVars(w io.Writer, data interface{}, vars map[string]interface{}) error {
	return t.execute(w, "", data, vars)
}

func (t *tmpl) ExecuteTemplateVars(w io.Writer, name string, data interface{}, vars map[string]interface{}) error {
	return t.execute(w, name, data, vars)
}

func (t *tmpl) replaceNode(n, p parse.Node, fname string) error {
	nn := &parse.VariableNode{
		NodeType: parse.NodeVariable,
		Pos:      n.Position(),
		Ident:    []string{"$Vars", "Ctx", fname},
	}
	return templateutil.ReplaceNode(n, p, nn)
}

func (t *tmpl) rewriteTranslationFuncs() error {
	for _, tr := range t.tmpl.Trees() {
		if err := templateutil.ReplaceTranslatableBlocks(tr, "t"); err != nil {
			return err
		}
		var err error
		templateutil.WalkTree(tr, func(n, p parse.Node) {
			if err != nil {
				return
			}
			if n.Type() == parse.NodeIdentifier {
				id := n.(*parse.IdentifierNode)
				switch id.Ident {
				case "t":
					err = t.replaceNode(n, p, "T")
				case "tn":
					err = t.replaceNode(n, p, "Tn")
				case "tc":
					err = t.replaceNode(n, p, "Tc")
				case "tnc":
					err = t.replaceNode(n, p, "Tnc")
				}
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tmpl) prepare() error {
	if err := t.tmpl.Compile(); err != nil {
		return err
	}
	return nil
}

func nop() interface{} { return nil }

func init() {
	if profile.On {
		inDevServer = os.Getenv("GONDOLA_DEV_SERVER") != ""
		if inDevServer {
			t := newInternalTemplate(&App{})
			t.tmpl.Funcs(template.FuncMap{
				"_gondola_profile_info": getProfileInfo,
				"_gondola_internal_asset": func(arg string) string {
					return internalAssetsManager.URL(arg)
				},
			})
			if err := t.Parse("profile.html"); err != nil {
				panic(err)
			}
			profileHook = &template.Hook{Template: t.tmpl, Position: assets.Bottom}
		}
	}
}
