package app

import (
	"net/http"

	"golang.org/x/net/websocket"
)

// Handler is the function type used to satisfy a request
// (not necessarily HTTP) with a given *Context.
// Gondola uses Handler for responding to HTTP requests
// (via gnd.la/app.App), executing commands
// (via gnd.la/commands) and tasks (via gnd.la/task).
type Handler func(*Context)

// WebsocketHandler is the function type used to handle
// websocket requests via HTTP(S). See also App.HandleWebsocket.
type WebsocketHandler func(*Context, *websocket.Conn)

// HandlerOptions represent the different options which might be
// specified when registering a Handler in an App.
type HandlerOptions struct {
	// Name indicates the Handler's name, which might be used
	// to reverse it with Context.Reverse of the "reverse"
	// template function.
	Name string
	// Host specifies the host the Handler will match. If non-empty,
	// only requests to this specific host will match the Handler.
	Host string
}

// A HandlerOption represents a function which receives a
// HandlerOptions, modifies and returns them. They're used
// in App.Handle() to set the options for a given handler.
type HandlerOption func(HandlerOptions) HandlerOptions

// NamedHandler sets the HandlerOptions.Name field. See HandlerOptions
// for more information.
func NamedHandler(name string) HandlerOption {
	return func(opts HandlerOptions) HandlerOptions {
		opts.Name = name
		return opts
	}
}

// HostHandler sets the HandlerOptions.Host field. See HandlerOptions
// for more information.
func HostHandler(host string) HandlerOption {
	return func(opts HandlerOptions) HandlerOptions {
		opts.Host = host
		return opts
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
