package form

import (
	"gnd.la/app"
	"gnd.la/form"
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
