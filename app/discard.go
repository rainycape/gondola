package app

import (
	"net/http"
)

var discard = &discarder{}

// discarded is used by background context
// to avoid crashing if a background task
// writes to the context.
type discarder struct {
}

func (d *discarder) Header() http.Header {
	return make(http.Header)
}

func (d *discarder) Write(b []byte) (int, error) {
	return len(b), nil
}

func (d *discarder) WriteHeader(_ int) {
}
