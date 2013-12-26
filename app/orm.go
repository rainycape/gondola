package app

import (
	"gnd.la/orm"
)

// Orm is just a very thin wrapper around
// orm.Orm, which disables the Close method
// when running in production mode, since
// the App is always reusing the same ORM
// instance.
type Orm struct {
	*orm.Orm
	debug bool
}

// Close calls orm.Orm.Close() only when in
// debug mode. Otherwise is a noop.
func (o *Orm) Close() error {
	if o.debug {
		return o.Orm.Close()
	}
	return nil
}
