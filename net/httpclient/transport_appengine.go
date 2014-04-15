// +build appengine

package httpclient

import (
	"errors"
	"net/http"
	"time"

	"appengine"
	"appengine/urlfetch"
)

var (
	errNoContext    = errors.New("no Context or no *http.Request provided, can't create a Transport")
	contextFallback func() appengine.Context
)

type errorTransport struct {
}

func (t *errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errNoContext
}

func newRoundTripper(ctx Context, tr *transport) http.RoundTripper {
	var r *http.Request
	if ctx != nil {
		r = ctx.Request()
	}
	var c appengine.Context
	if r == nil {
		if contextFallback == nil {
			return &errorTransport{}
		}
		c = contextFallback()
	} else {
		c = appengine.NewContext(r)
	}
	return &urlfetch.Transport{Context: c}
}

func (t *transport) Deadline() time.Duration {
	if tr, ok := t.transport.(*urlfetch.Transport); ok {
		return tr.Deadline
	}
	return 0
}

func (t *transport) SetDeadline(deadline time.Duration) {
	if tr, ok := t.transport.(*urlfetch.Transport); ok {
		tr.Deadline = deadline
	}
}
