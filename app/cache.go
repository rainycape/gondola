package app

import (
	"gnd.la/cache"
)

// Cache is just a very thin wrapper around
// cache.Cache, which disables the Close method
// when running in production mode, since
// the App is always reusing the same Cache
// instance.
type Cache struct {
	*cache.Cache
	debug bool
}

// Close calls cache.Cache.Close() only when in
// debug mode. Otherwise it is a noop.
func (c *Cache) Close() error {
	if c.debug {
		return c.Cache.Close()
	}
	return nil
}