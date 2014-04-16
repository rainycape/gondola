// build !appengine

package app

import (
	"errors"

	"gnd.la/cache"

	"appengine"
)

var (
	errNoAppCache = errors.New("App.Cache() does not work on App Engine - use Context.Cache() instead")
)

type contextSetter interface {
	SetContext(appengine.Context)
}

func (app *App) cache() (*Cache, error) {
	return nil, errNoAppCache
}

func (c *Context) cache() *Cache {
	ca, err := cache.New(defaultCache)
	if err != nil {
		panic(err)
	}
	if conn, ok := ca.Connection().(contextSetter); ok {
		ctx := appengine.NewContext(c.R)
		conn.SetContext(ctx)
	}
	return &Cache{Cache: ca}
}
