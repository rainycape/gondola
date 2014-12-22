// +build !appengine

package app

import (
	"fmt"

	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/net/mail"
	"gnd.la/orm"
)

// Methods that need to be redefined on appengine

func (app *App) cache() (*cache.Cache, error) {
	if app.c == nil {
		var err error
		app.locked(func() {
			if app.c != nil {
				return
			}
			if app.parent != nil {
				app.c, err = app.parent.cache()
				return
			}
			app.c, err = cache.New(app.cfg.Cache)
		})
		if err != nil {
			return nil, err
		}
	}
	return app.c, nil
}

func (app *App) orm() (*orm.Orm, error) {
	if app.o == nil {
		if err := app.prepareOrm(); err != nil {
			return nil, err
		}
	}
	return app.o, nil
}

func (app *App) blobstore() (*blobstore.Blobstore, error) {
	if app.store == nil {
		var err error
		app.locked(func() {
			if app.store != nil {
				return
			}
			if app.parent != nil {
				app.store, err = app.parent.blobstore()
				return
			}
			bs := app.cfg.Blobstore
			if bs == nil {
				err = errNoDefaultBlobstore
				return
			}
			app.store, err = blobstore.New(bs)
		})
		if err != nil {
			return nil, err
		}
	}
	return app.store, nil
}

func (app *App) checkPort() error {
	if p := app.cfg.Port; p <= 0 {
		return fmt.Errorf("port %d is invalid, must be > 0", p)
	}
	return nil
}

func (c *Context) cache() *cache.Cache {
	if c.app.c == nil {
		if _, err := c.app.Cache(); err != nil {
			panic(err)
		}
	}
	return c.app.c
}

func (c *Context) orm() *orm.Orm {
	if c.app.o == nil {
		if _, err := c.app.orm(); err != nil {
			panic(err)
		}
	}
	return c.app.o
}

func (c *Context) blobstore() *blobstore.Blobstore {
	if c.app.store == nil {
		_, err := c.app.Blobstore()
		if err != nil {
			panic(err)
		}
	}
	return c.app.store
}

func (c *Context) prepareMessage(msg *mail.Message) {
	// nop except on GAE
}
