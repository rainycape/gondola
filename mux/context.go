package mux

import (
	"gondola/cache"
	"net/http"
	"reflect"
)

type ContextFinalizer func(*Context)

type Context struct {
	W             http.ResponseWriter
	R             *http.Request
	submatches    []string
	params        map[string]string
	c             *cache.Cache
	cached        bool
	fromCache     bool
	handlerName   string
	mux           *Mux
	customContext interface{}
	Data          interface{} /* Left to the user */
}

func (c *Context) Count() int {
	return len(c.submatches)
}

func (c *Context) IndexValue(idx int) string {
	if idx < len(c.submatches) {
		return c.submatches[idx]
	}
	return ""
}

func (c *Context) ParamValue(name string) string {
	return c.params[name]
}

func (c *Context) Cache() *cache.Cache {
	if c.c == nil {
		c.c = cache.NewDefault()
	}
	return c.c
}

func (c *Context) SetCached(b bool) {
	c.cached = b
}

func (c *Context) SetServedFromCache(b bool) {
	c.fromCache = b
}

func (c *Context) Cached() bool {
	return c.cached
}

func (c *Context) ServedFromCache() bool {
	return c.fromCache
}

func (c *Context) HandlerName() string {
	return c.handlerName
}

// Returns the Mux this Context originated from
func (c *Context) Mux() *Mux {
	return c.mux
}

// Returns the custom type context wrapped in
// an interface{}. Intended for use in templates
// e.g. {{ Ctx.C.MyCustomMethod }}
//
// For use in code it's better to use the function
// you previously defined as the context transformation for
// the mux e.g.
// function in your own code to avoid type assertions
// func Context(ctx *mux.Context) *mycontext {
//	return (*mycontext)(ctx)
// }
// ...
// mymux.SetContextTransform(Context)

func (c *Context) C() interface{} {
	if c.customContext == nil {
		if c.mux.contextTransform != nil {
			result := c.mux.contextTransform.Call([]reflect.Value{reflect.ValueOf(c)})
			c.customContext = result[0].Interface()
		} else {
			c.customContext = c
		}
	}
	return c.customContext
}

func (c *Context) Close() {
	if c.c != nil {
		c.c.Close()
		c.c = nil
	}
}
