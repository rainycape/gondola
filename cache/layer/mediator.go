package layer

import (
	"gondola/mux"
	"gondola/util"
	"net/http"
)

type Mediator interface {
	Skip(ctx *mux.Context) bool
	Key(ctx *mux.Context) string
	Cache(ctx *mux.Context, responseCode int, outgoingHeaders http.Header) bool
	Expires(ctx *mux.Context, responseCode int, outgoingHeaders http.Header) int
}

type SimpleMediator struct {
	SkipCookies []string
	Expiration  int
}

func (m *SimpleMediator) Skip(ctx *mux.Context) bool {
	if m := ctx.R.Method; m != "GET" && m != "HEAD" {
		return true
	}
	c := ctx.Cookies()
	for _, v := range m.SkipCookies {
		if c.Has(v) {
			return true
		}
	}
	return false
}

func (m *SimpleMediator) Key(ctx *mux.Context) string {
	return util.Md5([]byte(ctx.R.Method + ctx.R.URL.String()))
}

func (m *SimpleMediator) Cache(ctx *mux.Context, responseCode int, outgoingHeaders http.Header) bool {
	return responseCode == http.StatusOK
}

func (m *SimpleMediator) Expires(ctx *mux.Context, responseCode int, outgoingHeaders http.Header) int {
	return m.Expiration
}
