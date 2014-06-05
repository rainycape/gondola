// +build appengine

package app

import (
	"errors"

	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/net/mail"
	"gnd.la/orm"
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
	// When using GCSQL, there's no need for
	// an appengine.Context to connect to the
	// Orm backend, so the App can provide an *Orm.
	// This is useful for automatically calling
	// Initialize() on the Orm in App.Prepare().
	//
	// Don't document this fact, just in case this
	// behavior changes in the future. Finding another
	// way to automatically call Orm.Initialize() is easy,
	// but breaking documented, public facing APIs is
	// just ugly.
	if o := app.o; o != nil {
		return o, nil
	}
	db := app.cfg.Database
	if db == nil || db.Scheme == "datastore" {
		return nil, errNoAppOrm
	}
	// Using GCSQL
	o, err := app.openOrm()
	if err != nil {
		return nil, err
	}
	return app.setOrm(o), nil
}

func (app *App) setOrm(o *orm.Orm) *Orm {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.o == nil {
		app.o = &Orm{Orm: o}
	} else {
		// Another goroutine set the ORM before us, just close
		// this one.
		o.Close()
	}
	return app.o
}

func (app *App) blobstore() (*blobstore.Blobstore, error) {
	return nil, errNoAppBlobstore
}

func (app *App) checkPort() error {
	return nil
}

func (c *Context) cache() *Cache {
	ca, err := cache.New(c.app.cfg.Cache)
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
	return c.app.setOrm(o)
}

func (c *Context) blobstore() *blobstore.Blobstore {
	bs := c.app.cfg.Blobstore
	if bs == nil {
		panic(errNoDefaultBlobstore)
	}
	b, err := blobstore.New(bs)
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
