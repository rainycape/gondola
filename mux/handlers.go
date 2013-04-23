package mux

func TemplateHandler(name string) Handler {
	return func(ctx *Context) {
		ctx.MustExecute(name, nil)
	}
}
