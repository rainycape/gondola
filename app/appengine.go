// +build appengine

package app

import (
	"errors"

	"gnd.la/cache"
	"gnd.la/net/mail"
	"gnd.la/orm/driver/datastore"

	"appengine"
)

var (
	errNoAppCache = errors.New("App.Cache() does not work on App Engine - use Context.Cache() instead")
	errNoAppOrm   = errors.New("App.Orm() does not work on App Engine - use Context.Orm() instead")
)

type contextSetter interface {
	SetContext(appengine.Context)
}

func (app *App) cache() (*Cache, error) {
	return nil, errNoAppCache
}

func (app *App) orm() (*Orm, error) {
	return nil, errNoAppOrm
}

func (app *App) checkPort() error {
	return nil
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

func (c *Context) orm() *Orm {
	o, err := c.app.openOrm()
	if err != nil {
		panic(err)
	}
	if drv, ok := o.Driver().(*datastore.Driver); ok {
		drv.SetContext(appengine.NewContext(c.R))
	}
	return &Orm{Orm: o}
}

func (c *Context) prepareMessage(msg *mail.Message) {
	msg.Context = appengine.NewContext(c.R)
}
