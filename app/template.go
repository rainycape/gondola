package app

import (
	"fmt"
	"gnd.la/loaders"
	"gnd.la/template"
	"gnd.la/template/assets"
	"io"
	"net/http"
	"reflect"
)

var reservedVariables = []string{"Ctx", "Request", "Exports"}

type Template interface {
	Execute(w io.Writer, data interface{}) error
	ExecuteTemplate(w io.Writer, name string, data interface{}) error
	ExecuteVars(w io.Writer, data interface{}, vars map[string]interface{}) error
	ExecuteTemplateVars(w io.Writer, name string, data interface{}, vars map[string]interface{}) error
}

type TemplateProcessor func(*template.Template) (*template.Template, error)

type tmpl struct {
	*template.Template
	app *App
}

func (t *tmpl) reverse(name string, args ...interface{}) (string, error) {
	if t.app != nil {
		_, s, err := t.app.reverse(name, args)
		return s, err
	}
	return "", fmt.Errorf("can't reverse %s because the app is not available", name)
}

func newTemplate(app *App, loader loaders.Loader, manager *assets.Manager) *tmpl {
	t := &tmpl{}
	t.app = app
	t.Template = template.New(loader, manager)
	t.Template.Debug = app.debug
	t.Template.Funcs(template.FuncMap{
		"reverse": t.reverse,
	})
	return t
}

func newAppTemplate(app *App) *tmpl {
	return newTemplate(app, app.templatesLoader, app.assetsManager)
}

func internalAssetsManager() *assets.Manager {
	return assets.NewManager(appAssets, assetsPrefix)
}

func newInternalTemplate(app *App) *tmpl {
	return newTemplate(app, appAssets, internalAssetsManager())
}

func (t *tmpl) ParseVars(file string, vars template.VarMap) error {
	if vars != nil {
		for _, k := range reservedVariables {
			vars[k] = nil
		}
	}
	return t.Template.ParseVars(file, vars)
}

func (t *tmpl) Parse(file string) error {
	return t.ParseVars(file, nil)
}

func (t *tmpl) execute(w io.Writer, name string, data interface{}, vars template.VarMap) error {
	var context *Context
	var request *http.Request
	if context, _ = w.(*Context); context != nil {
		request = context.R
	}
	va := map[string]interface{}{
		"Ctx":     context,
		"Request": request,
		"Exports": t.app.Exports(),
	}
	for k, v := range t.app.templateVars {
		va[k] = v
	}
	for k, v := range vars {
		va[k] = v
	}
	if context != nil {
		in := []reflect.Value{reflect.ValueOf(context)}
		for k, v := range t.app.templateVarFuncs {
			if _, ok := va[k]; !ok {
				out := v.Call(in)
				if len(out) == 2 && !out[1].IsNil() {
					return out[1].Interface().(error)
				}
				va[k] = out[0].Interface()
			}
		}
	}
	return t.Template.ExecuteTemplateVars(w, name, data, va)
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
