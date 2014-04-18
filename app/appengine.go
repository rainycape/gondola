// +build appengine

package app

import (
	"errors"

	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/net/mail"
	"gnd.la/orm/driver/datastore"

	"appengine"
)

var (
	errNoAppCache     = errors.New("App.Cache() does not work on App Engine - use Context.Cache() instead")
	errNoAppOrm       = errors.New("App.Orm() does not work on App Engine - use Context.Orm() instead")
	errNoAppBlobstore = errors.New("App.Blobstore() does not work on App Engine - use Context.Blobstore() instead")
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

func (app *App) blobstore() (*blobstore.Store, error) {
	return nil, errNoAppBlobstore
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
	// When using GCSQL, the Orm can be shared
	// among all requests.
	if o := c.app.o; o != nil {
		return o
	}
	o, err := c.app.openOrm()
	if err != nil {
		panic(err)
	}
	if drv, ok := o.Driver().(*datastore.Driver); ok {
		drv.SetContext(appengine.NewContext(c.R))
		return &Orm{Orm: o}
	}
	// We're using GCSQL backend
	c.app.mu.Lock()
	defer c.app.mu.Unlock()
	if c.app.o == nil {
		c.app.o = &Orm{Orm: o}
	} else {
		// Another goroutine set the ORM before us, just close
		// this one.
		o.Close()
	}
	return c.app.o
}

func (c *Context) blobstore() *blobstore.Store {
	if defaultBlobstore == nil {
		panic(errNoDefaultBlobstore)
	}
	b, err := blobstore.New(defaultBlobstore)
	if err != nil {
		panic(err)
	}
	if drv, ok := b.Driver().(contextSetter); ok {
		drv.SetContext(appengine.NewContext(c.R))
	}
	return b
}

func (c *Context) prepareMessage(msg *mail.Message) {
	msg.Context = appengine.NewContext(c.R)
}
