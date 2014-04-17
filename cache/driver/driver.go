package driver

import (
	"gnd.la/config"
)

var (
	drivers = map[string]Opener{}
)

// Opener is a function which returns a new Driver connection
// with the given driver-dependent value and the given options
// Some options are used by cache system and are documented in
// the Gondola's cache package, while others are driver dependent
// and are documented in each driver. Drivers should just ignore
// options which they don't use.
type Opener func(url *config.URL) (Driver, error)

// Driver is the interface implemented by cache drivers.
type Driver interface {
	// Set sets the cached value to the given []byte. Drivers should
	// interpret timeout as the number of seconds until the item
	// expires, with zero meaning no expiration.
	Set(key string, b []byte, timeout int) error
	// Get returns the []byte assocciated with the given key. If the
	// key is not found, no error should be returned. A nil []byte
	// with no error should be returned instead.
	Get(key string) ([]byte, error)
	// GetMulti returns the values for the given keys (if present) as a map.
	// The returned map should not include keys for keys not found in the cache.
	// As in Get(), an error should be returned only if there was an error when
	// communicating with the cache, not if no items (or just some items) were found.
	GetMulti(keys []string) (map[string][]byte, error)
	// Delete removes the given key from the cache. Drivers shouldn't
	// return an error if the key doesn't exists, only if there was
	// an error while deleting it.
	Delete(key string) error
	// Close closes the connection with the cache
	// backend.
	Close() error
	// Connection returns the underlying connection
	// to the cache, which is driver dependant and
	// might even be nil.
	Connection() interface{}
}

// Register registers a new cache driver with the
// given protocol and opener function. This function
// is not thread safe, as it's only intended to be
// used from the main goroutine.
func Register(name string, f Opener) {
	drivers[name] = f
}

// Get returns the opener function for the driver with
// the given name, or nil if there's no such driver.
func Get(name string) Opener {
	return drivers[name]
}
