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
	errNoAppBlobstore = errors.New("App.Blobstore() does not work on App Engine - use Context.Blobstore() instead")
)

type contextSetter interface {
	SetContext(appengine.Context)
}

func (app *App) cache() (*cache.Cache, error) {
	return nil, errNoAppCache
}

func (app *App) orm() (*orm.Orm, error) {
	// When using GCSQL, there's no need for
	// an appengine.Context to connect to the
	// Orm backend, so the App can provide an *Orm.
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
	if err := app.prepareOrm(); err != nil {
		return nil, err
	}
	return app.o, nil
}

func (app *App) blobstore() (*blobstore.Blobstore, error) {
	return nil, errNoAppBlobstore
}

func (app *App) checkPort() error {
	return nil
}

func (c *Context) cache() *cache.Cache {
	ca, err := cache.New(c.app.cfg.Cache)
	if err != nil {
		panic(err)
	}
	if conn, ok := ca.Connection().(contextSetter); ok {
		ctx := appengine.NewContext(c.R)
		conn.SetContext(ctx)
	}
	return ca
}

func (c *Context) orm() *orm.Orm {
	// When using GCSQL, the Orm can be shared
	// among all requests and must have been
	// initialized from App.Prepare
	if o := c.app.o; o != nil {
		return o
	}
	o, err := c.app.openOrm()
	if err != nil {
		panic(err)
	}
	if drv, ok := o.Driver().(*datastore.Driver); ok {
		drv.SetContext(appengine.NewContext(c.R))
	}
	return o
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
