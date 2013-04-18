package mux

import (
	"fmt"
	"gondola/files"
	"gondola/template"
	"net/http"
	"sync"
)

type Template struct {
	*template.Template
	mutex   sync.Mutex
	mux     *Mux
	context *Context
}

func newTemplate() *Template {
	t := &Template{}
	t.Template = template.New()
	t.Template.Funcs(template.FuncMap{
		"context": makeContext(t),
		"reverse": makeReverse(t),
		"request": makeRequest(t),
		"asset":   makeAsset(t),
	})
	return t
}

func (t *Template) Execute(ctx *Context, data interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.context = ctx
	return t.Template.Execute(ctx, data)
}

// Other functions which are defined depending on the template
// (because they require access to the context or the mux)
// context
// reverse
// request
// asset

func makeContext(t *Template) func() interface{} {
	return func() interface{} {
		return t.context
	}
}

func makeReverse(t *Template) func(string, ...interface{}) (string, error) {
	return func(name string, args ...interface{}) (string, error) {
		if t.mux != nil {
			return t.mux.Reverse(name, args...)
		}
		panic(fmt.Errorf("Can't reverse %s because the mux is not available", name))
	}
}

func makeRequest(t *Template) func() *http.Request {
	return func() *http.Request {
		if t.context != nil {
			return t.context.R
		}
		return nil
	}
}

func makeAsset(t *Template) func(string) string {
	return func(asset string) string {
		return files.StaticFileUrl(t.mux.staticFilesPrefix, asset)
	}
}
