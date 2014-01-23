package app

import (
	"gnd.la/cache"
)

// Cache is just a very thin wrapper around
// cache.Cache, which disables the Close method.
type Cache struct {
	*cache.Cache
}

// Close is a no-op. It prevents the App shared
// cache.Cache from being closed.
func (c *Cache) Close() error {
	return nil
}
