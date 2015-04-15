// +build !appengine

package app

import "gnd.la/log"

func (c *Context) logger() log.Interface {
	if c == nil || c.app.Logger == nil {
		return nullLogger{}
	}
	return c.app.Logger
}
