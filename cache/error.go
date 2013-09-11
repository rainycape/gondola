package cache

import (
	"fmt"
)

// cError is the interface implemented by some errors returned by the cache
// by a Cache instance. Don't expose this for now.
type cError interface {
	error
	// Key returns they key that was being retrieved or set. For performance
	// reasons, it might be empty if the error is ErrNotFound.
	Key() string
	// Codec returns true iff the error was caused by encoding or decoding
	Codec() bool
	// Err returns original error. Might be nil when error is ErrNotFound
	Err() error
}

type cacheError struct {
	op    string
	key   string
	codec bool
	err   error
}

func (c *cacheError) Error() string {
	return fmt.Sprintf("error %s (key %q): %s", c.op, c.key, c.err)
}

func (c *cacheError) Key() string {
	return c.key
}

func (c *cacheError) Codec() bool {
	return c.codec
}

func (c *cacheError) Err() error {
	return c.err
}
