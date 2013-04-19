package mux

import (
	"fmt"
	"gondola/files"
	"gondola/template"
	"io"
	"net/http"
)

type Template interface {
	Execute(w io.Writer, data interface{}) error
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

func (t *tmpl) Parse(file string) error {
	return t.Template.ParseVars(file, []string{"Context", "Request"})
}

func (t *tmpl) Execute(w io.Writer, data interface{}) error {
	var context *Context
	var request *http.Request
	if context, _ = w.(*Context); context != nil {
		request = context.R
	}
	vars := map[string]interface{}{"Context": context, "Request": request}
	return t.Template.ExecuteVars(w, data, vars)
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
