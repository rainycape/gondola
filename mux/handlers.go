package mux

func TemplateHandler(name string) Handler {
	return func(ctx *Context) {
		ctx.MustExecute(name, nil)
	}
}

func RedirectHandler(destination string, permanent bool) Handler {
	return func(ctx *Context) {
		ctx.Redirect(destination, permanent)
	}
}
