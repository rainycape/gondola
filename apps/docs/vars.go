package docs

import (
	"gnd.la/app"
	"gnd.la/apps/docs/doc"
	"gnd.la/signal"
)

var (
	GOROOT string
	GOPATH string
)

func init() {
	signal.Listen(app.WILL_LISTEN, func() {
		if GOROOT != "" {
			doc.Context.GOROOT = GOROOT
		}
		if GOPATH != "" {
			doc.Context.GOPATH = GOPATH
		}
	})
}
