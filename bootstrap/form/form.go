package form

import (
	"gnd.la/app"
	"gnd.la/form"
)

const (
	// NotChosen is an alias form form.NotChosen, to
	// avoid importing both packages.
	NotChosen = form.NotChosen
)

var (
	// Choose is an alias form form.Choose, to
	// avoid importing both packages.
	Choose = form.Choose
)

type Form struct {
	*form.Form
}

func (f *Form) Renderer() *Renderer {
	return f.Form.Renderer().(*Renderer)
}

func New(ctx *app.Context, opt *Options, values ...interface{}) *Form {
	r := &Renderer{}
	return &Form{form.New(ctx, r, (*form.Options)(opt), values...)}
}
