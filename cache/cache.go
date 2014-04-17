package cache

import (
	"errors"
	"fmt"
	"gnd.la/app/profile"
	"gnd.la/cache/driver"
	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/encoding/pipe"
	"gnd.la/log"
	"reflect"
	"strings"
)

var (
	ErrNotFound = errors.New("item not found in cache")
	imports     = map[string]string{
		"memcache": "gnd.la/cache/driver/memcache",
		"redis":    "gnd.la/cache/driver/redis",
	}
)

const cache = "cache"

type Cache struct {
	// The Logger to log debug messages and, more importantly, errors.
	// New() initialies the log.Logger to log.Std.
	Logger    *log.Logger
	prefix    string
	prefixLen int
	driver    driver.Driver
	codec     *codec.Codec
	pipe      *pipe.Pipe
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

// GetMulti returns several objects with only one trip to the cache.
// The queried keys are the ones set in the out parameter. Any key present
// in out and not found when querying the cache, will be deleted. The initial
// values in the out map should be any value of the type which will be used
// to decode the data for the given key.
// e.g. Let's suppose our cache holds 3 objects: k1 of type []int, k2 of type
// float64 and k3 of type string. To query for these 3 objects using GetMulti
// you would initialze out like this:
//
//  out := map[string]interface{}{
//	k1: []int(nil),
//	k2: float(0),
//	k3: "",
//  }
//  err := c.GetMulti(out, nil)
//
// After the GetMulti() call, any keys not present in the cache will be
// deleted from out, so len(out) will be <= 3 in this example.
//
// Alternatively, the second argument might be used to specify the types
// of the object to be decoded. Users might implement their own Typer
// or use UniTyper when requesting several objects of the same type.
func (c *Cache) GetMulti(out map[string]interface{}, typer Typer) error {
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	if profile.On {
		defer profile.Startf(cache, "GET MULTI %s", keys).End()
	}
	qkeys := keys
	if c.prefixLen > 0 {
		qkeys = make([]string, len(keys))
		for ii, v := range keys {
			qkeys[ii] = c.backendKey(v)
		}
	}
	data, err := c.driver.GetMulti(qkeys)
	if err != nil {
		gerr := &cacheError{
			op:  "getting multiple keys",
			key: strings.Join(keys, ", "),
			err: err,
		}
		c.error(gerr)
		return gerr
	}
	if typer == nil {
		typer = mapTyper(out)
	}
	for ii, k := range keys {
		value := data[qkeys[ii]]
		if value == nil {
			delete(out, k)
			continue
		}
		typ := typer.Type(k)
		if typ == nil {
			derr := &cacheError{
				op:  "determining output type for object",
				key: k,
				err: fmt.Errorf("untyped value for key %q - please, set the value type like e.g. out[%q] = ([]int)(nil) or out[%q] = float64(0)", k, k, k),
			}
			c.error(derr)
			return derr
		}
		val := reflect.New(typ)
		err := c.codec.Decode(value, val.Interface())
		if err != nil {
			derr := &cacheError{
				op:  "decoding object",
				key: k,
				err: err,
			}
			c.error(derr)
			return derr
		}
		out[k] = val.Elem().Interface()
	}
	return nil
}

// SetBytes stores the given byte array assocciated with
// the given key. See the documentation for Set for an
// explanation of the timeout parameter
func (c *Cache) SetBytes(key string, b []byte, timeout int) error {
	if profile.On {
		defer profile.Startf(cache, "SET %s", key).End()
	}
	if c.pipe != nil {
		var err error
		b, err = c.pipe.Encode(b)
		if err != nil {
			perr := &cacheError{
				op:  "encoding data with pipe",
				key: key,
				err: err,
			}
			c.error(perr)
			return perr
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
	if profile.On {
		defer profile.Startf(cache, "GET %s", key).End()
	}
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
	if c.pipe != nil {
		b, err = c.pipe.Decode(b)
		if err != nil {
			perr := &cacheError{
				op:  "decoding data with pipe",
				key: key,
				err: err,
			}
			c.error(perr)
			return nil, perr
		}
	}
	return b, nil
}

// Delete removes the key from the cache. An error is returned only
// if the item was found but couldn't be deleted. Deleting a non-existant
// item is always successful.
func (c *Cache) Delete(key string) error {
	if profile.On {
		defer profile.Startf(cache, "DELETE %s", key).End()
	}
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

// Close closes the cache connection. If you're using a cache
// using app.Context helper methods, the cache will be closed
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
	cache := &Cache{
		Logger: log.Std,
	}

	if codecName := conf.Fragment.Get("codec"); codecName != "" {
		cache.codec = codec.Get(codecName)
		if cache.codec == nil {
			if imp := codec.RequiredImport(codecName); imp != "" {
				return nil, fmt.Errorf("please import %q to use the codec %q", imp, codecName)
			}
			return nil, fmt.Errorf("unknown codec %q, maybe you forgot an import?", codecName)
		}
	} else {
		cache.codec = codec.Get("gob")
	}

	cache.prefix = conf.Fragment.Get("prefix")
	cache.prefixLen = len(cache.prefix)
	if pipeName := conf.Fragment.Get("pipe"); pipeName != "" {
		cache.pipe = pipe.Get(pipeName)
		if cache.pipe == nil {
			if imp := pipe.RequiredImport(pipeName); imp != "" {
				return nil, fmt.Errorf("please import %q to use the pipe %q", imp, pipeName)
			}
			return nil, fmt.Errorf("unknown pipe %q, maybe you forgot an import?", pipeName)
		}
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
	if cache.driver, err = opener(conf); err != nil {
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
