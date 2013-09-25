package form

import (
	"gnd.la/form"
	"gnd.la/mux"
)

type Form struct {
	*form.Form
}

func (f *Form) Renderer() *Renderer {
	return f.Form.Renderer().(*Renderer)
}

func New(ctx *mux.Context, opt *Options, values ...interface{}) *Form {
	r := &Renderer{}
	return &Form{form.New(ctx, r, (*form.Options)(opt), values...)}
}
