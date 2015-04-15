// +build appengine

package app

import (
	"fmt"

	"gnd.la/internal"
	"gnd.la/log"

	"appengine"
)

// gaeLoggger logs using the GAE logging APIs
type gaeLogger struct {
	c appengine.Context
}

func (g *gaeLogger) Debug(args ...interface{})                 { g.c.Debugf("%s", fmt.Sprint(args...)) }
func (g *gaeLogger) Debugf(format string, args ...interface{}) { g.c.Debugf(format, args...) }

func (g *gaeLogger) Info(args ...interface{})                 { g.c.Infof("%s", fmt.Sprint(args...)) }
func (g *gaeLogger) Infof(format string, args ...interface{}) { g.c.Infof(format, args...) }

func (g *gaeLogger) Warning(args ...interface{})                 { g.c.Warningf("%s", fmt.Sprint(args...)) }
func (g *gaeLogger) Warningf(format string, args ...interface{}) { g.c.Warningf(format, args...) }

func (g *gaeLogger) Error(args ...interface{})                 { g.c.Errorf("%s", fmt.Sprint(args...)) }
func (g *gaeLogger) Errorf(format string, args ...interface{}) { g.c.Errorf(format, args...) }

func (c *Context) logger() log.Interface {
	if c == nil || c.R == nil {
		return nullLogger{}
	}
	if internal.InAppEngineDevServer() {
		// Return the stderr logger here, since otherwise
		// the messages are not logged.
		if c.app.Logger == nil {
			return nullLogger{}
		}
		return c.app.Logger
	}
	return &gaeLogger{c: appengine.NewContext(c.R)}
}
