package mux

import (
	"gondola/cache"
)

type ContextFinalizer func(*Context)

type Context struct {
	submatches []string
	params     map[string]string
	c          *cache.Cache
	cached     bool
	fromCache  bool
	Data       interface{} /* Left to the user */
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

func (c *Context) Close() {
	if c.c != nil {
		c.c.Close()
		c.c = nil
	}
}
