// +build !appengine

package app

import (
	"gnd.la/cache"
	"gnd.la/log"
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
			if log.Std.Level() == log.LDebug {
				app.o.SetLogger(log.Std)
			}
		}
	}
	return app.o, nil
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

func (c *Context) prepareMessage(msg *mail.Message) {
	// nop except on GAE
}