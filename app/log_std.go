// +build !appengine

package app

import "gnd.la/log"

func (c *Context) logger() log.Interface {
	if c.app.Logger == nil {
		return nullLogger{}
	}
	return c.app.Logger
}
