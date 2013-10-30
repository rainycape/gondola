package cache

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"gnd.la/cache/codec"
	"gnd.la/cache/driver"
	"gnd.la/config"
	"gnd.la/log"
	"io"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrNotFound = errors.New("item not found in cache")
	imports     = map[string]string{
		"memcache": "gnd.la/cache/driver/memcache",
		"redis":    "gnd.la/cache/driver/redis",
		"msgpack":  "gnd.la/cache/codec/msgpack",
	}
)

type Cache struct {
	// The Logger to log debug messages and, more importantly, errors.
	// New() initialies the Logger to log.Std.
	Logger            *log.Logger
	prefix            string
	prefixLen         int
	driver            driver.Driver
	codec             *codec.Codec
	minCompressLength int
	compressLevel     int
	numQueries        int
}

func (c *Cache) backendKey(key string) string {
	if c.prefixLen > 0 {
		return c.prefix + key
	}
	return key
}

func (c *Cache) frontendKey(key string) string {
	return key[c.prefixLen:]
}

// Set stores the given object in the cache associated with the
// given key. Timeout is the number of seconds until the item
// expires. If the timeout is 0, the item never expires, but
// might be only purged from cache when running out of space.
func (c *Cache) Set(key string, object interface{}, timeout int) error {
	b, err := c.codec.Encode(object)
	if err != nil {
		eerr := &cacheError{
			op:    "encoding object",
			key:   key,
			codec: true,
			err:   err,
		}
		c.error(eerr)
		return eerr
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
	cerr := c.codec.Decode(b, obj)
	if cerr != nil {
		derr := &cacheError{
			op:    "decoding object",
			key:   key,
			codec: true,
			err:   cerr,
		}
		c.error(derr)
		return derr
	}
	return nil
}

// GetMulti returns several objects as a map[string]interface{}
// in only one roundtrip to the cache. The obj parameter is used only
// to initialize the objects before they're passed to the codec for decoding,
// since not all codecs include the object type in its serialization (e.g. JSON).
// If you're using a codec which encodes the object type (e.g. Gob) or you want
// to decode the objects into a raw interface{} you might pass nil as the second
// parameter to this function.
func (c *Cache) GetMulti(keys []string, obj interface{}) (map[string]interface{}, error) {
	c.numQueries++
	if c.prefixLen > 0 {
		k := make([]string, len(keys))
		for ii, v := range keys {
			k[ii] = c.backendKey(v)
		}
		keys = k
	}
	data, err := c.driver.GetMulti(keys)
	if err != nil {
		gerr := &cacheError{
			op:  "getting multiple keys",
			key: strings.Join(keys, ", "),
			err: err,
		}
		c.error(gerr)
		return nil, gerr
	}
	objects := make(map[string]interface{}, len(data))
	var typ *reflect.Type
	if obj != nil {
		t := reflect.TypeOf(obj)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		typ = &t
	}
	for k, v := range data {
		var object interface{}
		if typ != nil {
			p := reflect.New(*typ)
			err = c.codec.Decode(v, p.Interface())
			object = p.Elem().Interface()
		} else {
			err = c.codec.Decode(v, &object)
		}
		if err != nil {
			derr := &cacheError{
				op:  "decoding object",
				key: k,
				err: err,
			}
			c.error(derr)
			return nil, derr
		}
		objects[c.frontendKey(k)] = object
	}
	return objects, nil
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
					c.debugf("Compressed key %s from %d bytes to %d bytes", key, l, cl)
				}
			} else {
				c.warningf("error opening zlib writer: %s", err)
			}
		}
	}
	k := c.backendKey(key)
	err := c.driver.Set(k, b, timeout)
	if err != nil {
		serr := &cacheError{
			op:  "setting key",
			key: key,
			err: err,
		}
		c.error(serr)
		return serr
	}
	c.debugf("Set key %s (%d bytes), expiring in %d", k, len(b), timeout)
	return nil
}

// GetBytes returns the byte array assocciated with the given key
func (c *Cache) GetBytes(key string) ([]byte, error) {
	c.numQueries++
	b, err := c.driver.Get(c.backendKey(key))
	if err != nil {
		gerr := &cacheError{
			op:  "getting key",
			key: key,
			err: err,
		}
		c.error(gerr)
		return nil, gerr
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
		derr := &cacheError{
			op:  "deleting",
			key: key,
			err: err,
		}
		c.error(derr)
		return derr
	}
	return nil
}

// NumQueries returns the number of queries made to this
// cache since it was initialized.
func (c *Cache) NumQueries() int {
	return c.numQueries
}

// Close closes the cache connection. If you're using a cache
// using mux.Context helper methods, the cache will be closed
// for you.
func (c *Cache) Close() error {
	return c.driver.Close()
}

// Connection returns a interface{} wrapping the native connection
// type for the cache client (e.g. a memcache or redis connection).
// Some drivers might return a nil connection (like the fs or the
// dummy driver).
func (c *Cache) Connection() interface{} {
	return c.driver.Connection()
}

func (c *Cache) debugf(format string, arg ...interface{}) {
	if c.Logger != nil {
		c.Logger.Debugf(format, arg...)
	}
}

func (c *Cache) warningf(format string, arg ...interface{}) {
	if c.Logger != nil {
		c.Logger.Warningf(format, arg...)
	}
}

func (c *Cache) error(err *cacheError) {
	if c.Logger != nil {
		c.Logger.Error(err)
	}
}

func newConfig(conf *config.URL) (*Cache, error) {
	if conf.Options == nil {
		conf.Options = config.Options{}
	}
	cache := &Cache{
		Logger:            log.Std,
		minCompressLength: -1,
		compressLevel:     zlib.DefaultCompression,
	}

	if codecName := conf.Get("codec"); codecName != "" {
		cache.codec = codec.Get(codecName)
		if cache.codec == nil {
			if imp := imports[codecName]; imp != "" {
				return nil, fmt.Errorf("please import %q to use the cache codec %q", imp, codecName)
			}
			return nil, fmt.Errorf("unknown cache codec %q, maybe you forgot an import?", codecName)
		}
	} else {
		cache.codec = codec.Get("gob")
	}

	cache.prefix = conf.Get("prefix")
	cache.prefixLen = len(cache.prefix)

	if mcl := conf.Get("min_compress"); mcl != "" {
		val, err := strconv.Atoi(mcl)
		if err != nil {
			return nil, fmt.Errorf("invalid min_compress value %q (%s) (must be an integer)", mcl, err)
		}
		cache.minCompressLength = val
	}

	if cl := conf.Get("compress_level"); cl != "" {
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
	if conf.Scheme != "" {
		opener = driver.Get(conf.Scheme)
		if opener == nil {
			if imp := imports[conf.Scheme]; imp != "" {
				return nil, fmt.Errorf("please import %q to use the cache driver %q", imp, conf.Scheme)
			}
			return nil, fmt.Errorf("unknown cache driver %q, maybe you forgot an import?", conf.Scheme)
		}
	} else {
		opener = driver.Get("dummy")
	}
	var err error
	if cache.driver, err = opener(conf.Value, conf.Options); err != nil {
		return nil, err
	}
	return cache, nil
}

// New returns a new cache instance, using the given
// configuration URL. If the configuration is nil, a
// dummy cache is returned, which always returns that the
// object does not exists in the cache.
func New(url *config.URL) (*Cache, error) {
	if url == nil {
		// Use dummy cache
		url = &config.URL{}
	}
	return newConfig(url)
}
