// +build appengine

package httpclient

import (
	"runtime"

	"appengine"
	"appengine/aetest"
)

func aetestContext() appengine.Context {
	c, err := aetest.NewContext(nil)
	if err != nil {
		panic(err)
	}
	// Not exactly perfect, since some Context might
	// fail be closed after tests.
	runtime.SetFinalizer(c, func(ctx aetest.Context) {
		ctx.Close()
	})
	return c
}

func init() {
	contextFallback = aetestContext
}
