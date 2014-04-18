// +build !appengine

package app

import (
	"fmt"

	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/net/mail"
)

// Methods that need to be redefined on appengine

func (app *App) cache() (*Cache, error) {
	if app.c == nil {
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.c == nil {
			if app.parent != nil {
				var err error
				app.c, err = app.parent.Cache()
				if err != nil {
					return nil, err
				}
			} else {
				c, err := cache.New(defaultCache)
				if err != nil {
					return nil, err
				}
				app.c = &Cache{Cache: c}
			}
		}
	}
	return app.c, nil
}

func (app *App) orm() (*Orm, error) {
	if app.o == nil {
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.o == nil {
			if app.parent != nil {
				var err error
				app.o, err = app.parent.Orm()
				if err != nil {
					return nil, err
				}
			} else {
				o, err := app.openOrm()
				if err != nil {
					return nil, err
				}
				app.o = &Orm{Orm: o}
			}
		}
	}
	return app.o, nil
}

func (app *App) blobstore() (*blobstore.Store, error) {
	if app.store == nil {
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.store == nil {
			var err error
			if app.parent != nil {
				app.store, err = app.parent.Blobstore()
			} else {
				if defaultBlobstore == nil {
					return nil, errNoDefaultBlobstore
				}
				app.store, err = blobstore.New(defaultBlobstore)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return app.store, nil
}

func (app *App) checkPort() error {
	if app.Port <= 0 {
		return fmt.Errorf("port %d is invalid, must be > 0", app.Port)
	}
	return nil
}

func (c *Context) cache() *Cache {
	if c.app.c == nil {
		if _, err := c.app.Cache(); err != nil {
			panic(err)
		}
	}
	return c.app.c
}

func (c *Context) orm() *Orm {
	if c.app.o == nil {
		if _, err := c.app.Orm(); err != nil {
			panic(err)
		}
	}
	return c.app.o
}

func (c *Context) blobstore() *blobstore.Store {
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
