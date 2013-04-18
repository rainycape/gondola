package mux

import (
	"fmt"
	"gondola/files"
	"gondola/template"
	"net/http"
	"sync"
	"io"
)

type Template interface {
    Execute(w io.Writer, data interface{}) error
}

type tmpl struct {
	*template.Template
	mutex   sync.Mutex
	mux     *Mux
	context *Context
}

func newTemplate() *tmpl {
	t := &tmpl{}
	t.Template = template.New()
	t.Template.Funcs(template.FuncMap{
		"context": makeContext(t),
		"reverse": makeReverse(t),
		"request": makeRequest(t),
		"asset":   makeAsset(t),
	})
	return t
}

func (t *tmpl) Execute(w io.Writer, data interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.context, _ = w.(*Context)
	return t.Template.Execute(w, data)
}

// Other functions which are defined depending on the template
// (because they require access to the context or the mux)
// context
// reverse
// request
// asset

func makeContext(t *tmpl) func() interface{} {
	return func() interface{} {
		return t.context
	}
}

func makeReverse(t *tmpl) func(string, ...interface{}) (string, error) {
	return func(name string, args ...interface{}) (string, error) {
		if t.mux != nil {
			return t.mux.Reverse(name, args...)
		}
		panic(fmt.Errorf("Can't reverse %s because the mux is not available", name))
	}
}

func makeRequest(t *tmpl) func() *http.Request {
	return func() *http.Request {
		if t.context != nil {
			return t.context.R
		}
		return nil
	}
}

func makeAsset(t *tmpl) func(string) string {
	return func(asset string) string {
		return files.StaticFileUrl(t.mux.staticFilesPrefix, asset)
	}
}
