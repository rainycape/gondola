package cache

import (
	"bytes"
	"compress/zlib"
	"gondola/cache/codec"
	"gondola/cache/driver"
	"gondola/log"
	"io"
	"strconv"
)

type Cache struct {
	Prefix            string
	Driver            driver.Driver
	Codec             *codec.Codec
	MinCompressLength int
	CompressLevel     int
	numQueries        int
}

func (c *Cache) manipulatesKeys() bool {
	return c.Prefix != ""
}

func (c *Cache) backendKey(key string) string {
	return c.Prefix + key
}

func (c *Cache) frontendKey(key string) string {
	if c.Prefix != "" {
		return key[len(c.Prefix):]
	}
	return key
}

// Set stores the given object in the cache associated with the
// given key. Timeout is the number of seconds until the item
// expires. If the timeout is 0, the item is never expired and
// might be only purged from cache when running out of space
func (c *Cache) Set(key string, object interface{}, timeout int) error {
	b, err := c.Codec.Encode(&object)
	if err != nil {
		log.Logf(c.errLevel(err), "Error encoding object for key %s: %s", key, err)
		return err
	}
	return c.SetBytes(key, b, timeout)
}

// Get returns the item assocciated with the given key, wrapped
// in an interface{}
func (c *Cache) Get(key string) interface{} {
	var obj interface{}
	if c.GetObject(key, &obj) {
		return obj
	}
	return nil
}

// GetObject returns the item associated with the given key
// using the passed in interface{} (which should be a pointer to
// the same struct type that was stored with Set). Returns true
// if the object could be succesfully decoded.
func (c *Cache) GetObject(key string, obj interface{}) bool {
	b := c.GetBytes(key)
	if b != nil {
		err := c.Codec.Decode(b, obj)
		if err == nil {
			return true
		}
		log.Logf(c.errLevel(err), "Error decoding object for key %s: %s", key, err)
	}
	return false
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
	data, err := c.Driver.GetMulti(keys)
	if err != nil {
		log.Logf(c.errLevel(err), "Error querying cache for keys %v: %v", keys, err)
		return nil
	}
	objects := make(map[string]interface{}, len(data))
	if c.manipulatesKeys() {
		for k, v := range data {
			var object interface{}
			err := c.Codec.Decode(v, &object)
			if err != nil {
				log.Logf(c.errLevel(err), "Error decoding object for key %s: %s", k, err)
				continue
			}
			objects[c.frontendKey(k)] = object
		}
	} else {
		for k, v := range data {
			var object interface{}
			err := c.Codec.Decode(v, &object)
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
	if c.MinCompressLength >= 0 {
		if l := len(b); l > c.MinCompressLength {
			var buf bytes.Buffer
			w, err := zlib.NewWriterLevel(&buf, c.CompressLevel)
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
	err := c.Driver.Set(c.backendKey(key), b, timeout)
	if err != nil {
		log.Logf(c.errLevel(err), "Error setting cache key %s: %s", key, err)
	}
	return err
}

// GetBytes returns the byte array assocciated with the given key
func (c *Cache) GetBytes(key string) []byte {
	c.numQueries++
	b, err := c.Driver.Get(c.backendKey(key))
	if err != nil {
		log.Errorf("Error getting cache key %s: %s", key, err)
		return nil
	}
	if c.MinCompressLength >= 0 && len(b) > 0 {
		r, err := zlib.NewReader(bytes.NewBuffer(b))
		if err == nil {
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			if err == nil {
				if r.Close() == nil {
					log.Debugf("Decompressed key %s", key)
					b = buf.Bytes()
				}
			} else {
				r.Close()
			}
		}
	}
	return b
}

// Delete removes the key from the cache
func (c *Cache) Delete(key string) error {
	c.numQueries++
	err := c.Driver.Delete(c.backendKey(key))
	if err != nil {
		log.Logf(c.errLevel(err), "Error deleting cache key %s: %s", key, err)
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
	c.Driver.Close()
}

// Connection returns a interface{} wrapping the native connection
// type for the cache client (e.g. a memcache or redis connection)
func (c *Cache) Connection() interface{} {
	return c.Driver.Connection()
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

func NewConfig(config *Config) *Cache {
	if config.Options == nil {
		config.Options = driver.Options{}
	}
	cache := &Cache{
		MinCompressLength: -1,
		CompressLevel:     zlib.DefaultCompression,
	}
	var c *codec.Codec
	if codecName := config.Get("codec"); codecName != "" {
		c = codec.Get(codecName)
		if c == nil {
			log.Warningf("Unknown cache codec name %q, using gob\n", codecName)
		}
	}
	cache.Prefix = config.Get("prefix")
	if mcl := config.Get("min_compress"); mcl != "" {
		val, err := strconv.Atoi(mcl)
		if err == nil {
			cache.MinCompressLength = val
		} else {
			log.Warningf("Invalid min_compress value %q (must be integer)", mcl)
		}
	}

	if cl := config.Get("compress_level"); cl != "" {
		val, err := strconv.Atoi(cl)
		if err == nil {
			if val >= zlib.NoCompression && val <= zlib.BestCompression {
				cache.CompressLevel = val
			} else {
				log.Warningf("Invalid compress_level value %d (must be between %d and %d)",
					val, zlib.NoCompression, zlib.BestCompression)
			}
		} else {
			log.Warningf("Invalid compress_level value %q (must be integer)", cl)
		}
	}
	if c == nil {
		c = codec.GobCodec
	}
	cache.Codec = c
	opener := driver.Get(config.Driver)
	if opener == nil {
		opener = driver.OpenDummyDriver
		if config.Driver != "" {
			log.Warningf("Unknown cache driver %q, using dummy.", config.Driver)
		}
	}
	cache.Driver = opener(config.Value, config.Options)
	return cache
}

func NewDefault() *Cache {
	return NewConfig(&defaultCache)
}

func New(config string) (*Cache, error) {
	cfg, err := ParseConfig(config)
	if err != nil {
		return nil, err
	}
	return NewConfig(cfg), nil
}
