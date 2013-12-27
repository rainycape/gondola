package app

import (
	"fmt"
	"net/url"
)

// Transformer is a function which receives a Handler and
// returns another handler, usually with some functionality
// added (e.g. requiring a signed in user, or adding a cache layer).
type Transformer func(Handler) Handler

// SignedIn returns a new Handler which requires a signed in
// user to be executed. If there's no signed in user, it returns
// a redirect to the handler named "sign-in", indicating the
// previous url in the "from" parameter. If there's no handler
// named "sign-in", it panics. It also adds "Cookie" to the Vary
// header, and "private" to the Cache-Control header.
func SignedIn(handler Handler) Handler {
	return func(ctx *Context) {
		h := ctx.Header()
		h.Add("Vary", "Cookie")
		h.Add("Cache-Control", "private")
		if ctx.User() == nil {
			signIn := ctx.MustReverse("sign-in")
			u, err := url.Parse(signIn)
			if err != nil {
				panic(err)
			}
			from := ctx.URL().String()
			u.RawQuery += fmt.Sprintf("%s=%s", SignInFromParameterName, url.QueryEscape(from))
			ctx.Redirect(u.String(), false)
			return
		}
		handler(ctx)
	}
}

// Anonymous returns a new handler which redirects signed in users
// to the previous page (or the root page if there's no referrer).
func Anonymous(handler Handler) Handler {
	return func(ctx *Context) {
		h := ctx.Header()
		h.Add("Vary", "Cookie")
		h.Add("Cache-Control", "private, must-revalidate")
		if ctx.User() != nil {
			ctx.RedirectBack()
			return
		}
		handler(ctx)
	}
}

// Headers returns a new Handler which adds the given headers
// to every response.
func Headers(handler Handler, headers Header) Handler {
	return func(ctx *Context) {
		h := ctx.Header()
		for k, v := range headers {
			for _, val := range v {
				h.Add(k, val)
			}
		}
		handler(ctx)
	}
}

// Vary returns a new Handler which adds the given
// values to the Vary header.
func Vary(handler Handler, values []string) Handler {
	return Headers(handler, Header{"Vary": values})
}

// Private returns a new Handler which adds the
// headers Vary: Cookie and Cache-Control: private.
func Private(handler Handler) Handler {
	return Headers(handler, Header{
		"Vary":          {"Cookie"},
		"Cache-Control": {"private"},
	})
}
