package mux

import (
	"fmt"
	"gondola/files"
	"gondola/template"
	"io"
	"net/http"
	"reflect"
)

var reservedVariables = []string{"Context", "Request"}

type Template interface {
	Execute(w io.Writer, data interface{}) error
	ExecuteVars(w io.Writer, data interface{}, vars map[string]interface{}) error
}

type tmpl struct {
	*template.Template
	mux *Mux
}

func newTemplate() *tmpl {
	t := &tmpl{}
	t.Template = template.New()
	t.Template.Funcs(template.FuncMap{
		"reverse": makeReverse(t),
		"asset":   makeAsset(t),
	})
	return t
}

func (t *tmpl) ParseVars(file string, vars []string) error {
	return t.Template.ParseVars(file, append(vars, reservedVariables...))
}

func (t *tmpl) execute(w io.Writer, data interface{}, vars map[string]interface{}) error {
	var context *Context
	var request *http.Request
	if context, _ = w.(*Context); context != nil {
		request = context.R
	}
	va := map[string]interface{}{
		"Context": context,
		"Request": request,
	}
	for k, v := range t.mux.templateVars {
		va[k] = v
	}
	for k, v := range vars {
		va[k] = v
	}
	if context != nil {
		in := []reflect.Value{reflect.ValueOf(context)}
		for k, v := range t.mux.templateVarFuncs {
			if _, ok := va[k]; !ok {
				out := v.Call(in)
				if len(out) == 2 && !out[1].IsNil() {
					return out[1].Interface().(error)
				}
				va[k] = out[0].Interface()
			}
		}
	}
	return t.Template.ExecuteVars(w, data, va)
}

func (t *tmpl) Execute(w io.Writer, data interface{}) error {
	return t.execute(w, data, nil)
}

func (t *tmpl) ExecuteVars(w io.Writer, data interface{}, vars map[string]interface{}) error {
	return t.execute(w, data, vars)
}

// Other functions which are defined depending on the template
// (because they require access to the context or the mux)
// reverse
// asset

func makeReverse(t *tmpl) func(string, ...interface{}) (string, error) {
	return func(name string, args ...interface{}) (string, error) {
		if t.mux != nil {
			return t.mux.Reverse(name, args...)
		}
		return "", fmt.Errorf("Can't reverse %s because the mux is not available", name)
	}
}

func makeAsset(t *tmpl) func(string) string {
	return func(asset string) string {
		return files.StaticFileUrl(t.mux.staticFilesPrefix, asset)
	}
}
