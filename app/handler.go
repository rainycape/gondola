package app

import "net/http"

// Handler is the function type used to satisfy a request
// (not necessarily HTTP) with a given *Context.
// Gondola uses Handler for responding to HTTP requests
// (via gnd.la/app.App), executing administrative commands
// (via gnd.la/admin) and tasks (via gnd.la/task).
type Handler func(*Context)

// Options represent the different options which might be
// specified when registering a Handler in an App.
type Options struct {
	// Name indicates the Handler's name, which might be used
	// to reverse it with Context.Reverse of the "reverse"
	// template function.
	Name string
	// Host specifies the host the Handler will match. If non-empty,
	// only requests to this specific host will match the Handler.
	Host string
}

type HandlerInfo struct {
	Handler Handler
	Options *Options
}

func NamedHandler(name string, handler Handler) *HandlerInfo {
	return &HandlerInfo{
		Handler: handler,
		Options: &Options{
			Name: name,
		},
	}
}

// HandlerFromHTTPFunc returns a Handler from an http.HandlerFunc.
func HandlerFromHTTPFunc(f http.HandlerFunc) Handler {
	return func(ctx *Context) {
		f(ctx, ctx.R)
	}
}

// HandlerFromHTTPHandler returns a Handler from an http.Handler.
func HandlerFromHTTPHandler(h http.Handler) Handler {
	return func(ctx *Context) {
		h.ServeHTTP(ctx, ctx.R)
	}
}

func includedAppHandler(app *App, prefix string) Handler {
	prefixLen := len(prefix)
	return func(ctx *Context) {
		prevApp := ctx.app
		defer func() {
			ctx.app = prevApp
		}()
		ctx.app = app
		defer func() {
			ctx.app = app
		}()
		app.serveOrNotFound(ctx.R.URL.Path[prefixLen:], ctx)
	}
}
