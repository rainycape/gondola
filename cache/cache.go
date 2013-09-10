package cache

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"gondola/cache/codec"
	"gondola/cache/driver"
	"gondola/log"
	"io"
	"strconv"
)

var (
	ErrNotFound = errors.New("item not found in cache")
)

type Cache struct {
	prefix            string
	driver            driver.Driver
	codec             *codec.Codec
	minCompressLength int
	compressLevel     int
	numQueries        int
}

func (c *Cache) manipulatesKeys() bool {
	return c.prefix != ""
}

func (c *Cache) backendKey(key string) string {
	return c.prefix + key
}

func (c *Cache) frontendKey(key string) string {
	if c.prefix != "" {
		return key[len(c.prefix):]
	}
	return key
}

// Set stores the given object in the cache associated with the
// given key. Timeout is the number of seconds until the item
// expires. If the timeout is 0, the item never expires, but
// might be only purged from cache when running out of space.
func (c *Cache) Set(key string, object interface{}, timeout int) error {
	b, err := c.codec.Encode(object)
	if err != nil {
		log.Logf(c.errLevel(err), "error encoding object for key %s: %s", key, err)
		return err
	}
	return c.SetBytes(key, b, timeout)
}

// Get retrieves the requested item from the cache and decodes it
// into the passed interface{}, which must be addressable.
// If the item is not found in the cache, ErrNotFound is returned.
// Other errors mean that either there was a problem communicating
// with the cache or the item could not be decoded.
func (c *Cache) Get(key string, obj interface{}) error {
	b, err := c.GetBytes(key)
	if err != nil {
		return err
	}
	err = c.codec.Decode(b, obj)
	if err != nil {
		log.Logf(c.errLevel(err), "error decoding object for key %s: %s", key, err)
		return err
	}
	return nil
}

// GetMulti returns several objects as a map[string]interface{}
// in only one roundtrip to the cache
func (c *Cache) GetMulti(keys []string) map[string]interface{} {
	c.numQueries++
	if c.manipulatesKeys() {
		k := make([]string, len(keys))
		for ii, v := range keys {
			k[ii] = c.backendKey(v)
		}
		keys = k
	}
	data, err := c.driver.GetMulti(keys)
	if err != nil {
		log.Logf(c.errLevel(err), "Error querying cache for keys %v: %v", keys, err)
		return nil
	}
	objects := make(map[string]interface{}, len(data))
	if c.manipulatesKeys() {
		for k, v := range data {
			var object interface{}
			err := c.codec.Decode(v, &object)
			if err != nil {
				log.Logf(c.errLevel(err), "Error decoding object for key %s: %s", k, err)
				continue
			}
			objects[c.frontendKey(k)] = object
		}
	} else {
		for k, v := range data {
			var object interface{}
			err := c.codec.Decode(v, &object)
			if err != nil {
				log.Logf(c.errLevel(err), "Error decoding object for key %s: %s", k, err)
				continue
			}
			objects[k] = object
		}
	}
	return objects
}

// SetBytes stores the given byte array assocciated with
// the given key. See the documentation for Set for an
// explanation of the timeout parameter
func (c *Cache) SetBytes(key string, b []byte, timeout int) error {
	c.numQueries++
	if c.minCompressLength >= 0 {
		if l := len(b); l > c.minCompressLength {
			var buf bytes.Buffer
			w, err := zlib.NewWriterLevel(&buf, c.compressLevel)
			if err == nil {
				w.Write(b)
				w.Close()
				cb := buf.Bytes()
				if cl := len(cb); cl < l {
					b = cb
					log.Debugf("Compressed key %s from %d bytes to %d bytes", key, l, cl)
				}
			} else {
				log.Warningf("Error opening zlib writer: %s", err)
			}
		}
	}
	log.Debugf("Setting key %s (%d bytes)", c.backendKey(key), len(b))
	err := c.driver.Set(c.backendKey(key), b, timeout)
	if err != nil {
		log.Logf(c.errLevel(err), "Error setting cache key %s: %s", key, err)
	}
	return err
}

// GetBytes returns the byte array assocciated with the given key
func (c *Cache) GetBytes(key string) ([]byte, error) {
	c.numQueries++
	b, err := c.driver.Get(c.backendKey(key))
	if err != nil {
		log.Errorf("error getting cache key %s: %s", key, err)
		return nil, err
	}
	if b == nil {
		return nil, ErrNotFound
	}
	if c.minCompressLength >= 0 && len(b) > 0 {
		r, err := zlib.NewReader(bytes.NewBuffer(b))
		if err == nil {
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			if err == nil {
				if r.Close() == nil {
					b = buf.Bytes()
				}
			} else {
				r.Close()
			}
		}
	}
	return b, nil
}

// Delete removes the key from the cache. An error is returned only
// if the item was found but couldn't be deleted. Deleting a non-existant
// item is always successful.
func (c *Cache) Delete(key string) error {
	c.numQueries++
	err := c.driver.Delete(c.backendKey(key))
	if err != nil {
		log.Logf(c.errLevel(err), "error deleting cache key %s: %s", key, err)
	}
	return err
}

// NumQueries returns the number of queries made to this
// cache since it was initialized.
func (c *Cache) NumQueries() int {
	return c.numQueries
}

// Close closes the cache connection. If you're using a cache
// using mux.Context helper methods, the cache will be closed
// for you.
func (c *Cache) Close() {
	c.driver.Close()
}

// Connection returns a interface{} wrapping the native connection
// type for the cache client (e.g. a memcache or redis connection).
// Some drivers might return a nil connection (like the fs or the
// dummy driver).
func (c *Cache) Connection() interface{} {
	return c.driver.Connection()
}

func (c *Cache) errLevel(err interface{}) log.LLevel {
	type temporarier interface {
		Temporary() bool
	}
	if terr, ok := err.(temporarier); ok && terr.Temporary() {
		return log.LWarning
	}
	return log.LError
}

func newConfig(config *config) (*Cache, error) {
	if config.Options == nil {
		config.Options = driver.Options{}
	}
	cache := &Cache{
		minCompressLength: -1,
		compressLevel:     zlib.DefaultCompression,
	}

	if codecName := config.Get("codec"); codecName != "" {
		cache.codec = codec.Get(codecName)
		if cache.codec == nil {
			return nil, fmt.Errorf("unknown cache codec %q, maybe you forgot an import?", codecName)
		}
	} else {
		cache.codec = codec.Get("gob")
	}

	cache.prefix = config.Get("prefix")

	if mcl := config.Get("min_compress"); mcl != "" {
		val, err := strconv.Atoi(mcl)
		if err != nil {
			return nil, fmt.Errorf("invalid min_compress value %q (%s) (must be an integer)", mcl, err)
		}
		cache.minCompressLength = val
	}

	if cl := config.Get("compress_level"); cl != "" {
		val, err := strconv.Atoi(cl)
		if err != nil {
			return nil, fmt.Errorf("invalid compress_level %q (%s) (must be an integer)", cl, err)
		}
		if (val < zlib.NoCompression || val > zlib.BestCompression) && val != -1 {
			return nil, fmt.Errorf("invalid compress_level %d (must be -1 or between %d and %d)",
				val, zlib.NoCompression, zlib.BestCompression)
		}
		cache.compressLevel = val
	}
	var opener driver.Opener
	if config.Driver != "" {
		opener = driver.Get(config.Driver)
		if opener == nil {
			return nil, fmt.Errorf("unknown cache driver %q, maybe you forgot an import?", config.Driver)
		}
	} else {
		opener = driver.Get("dummy")
	}
	var err error
	if cache.driver, err = opener(config.Value, config.Options); err != nil {
		return nil, err
	}
	return cache, nil
}

func New(config string) (*Cache, error) {
	cfg, err := parseConfig(config)
	if err != nil {
		return nil, err
	}
	return newConfig(cfg)
}
