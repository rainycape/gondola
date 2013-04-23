package cache

import (
	"bytes"
	"compress/zlib"
	"gondola/log"
	"io"
	"net/url"
	"strconv"
)

type BackendInitializer func(*url.URL) Backend

var (
	defaultCacheUrl string
	backends        = map[string]BackendInitializer{}
	codecs          = map[string]*Codec{}
)

type Backend interface {
	Set(key string, b []byte, timeout int) error
	Get(key string) ([]byte, error)
	GetMulti(keys []string) (map[string][]byte, error)
	Delete(key string) error
	Close() error
}

type Codec struct {
	Encode func(v interface{}) ([]byte, error)
	Decode func(data []byte, v interface{}) error
}

type Cache struct {
	Prefix            string
	Backend           Backend
	Codec             *Codec
	MinCompressLength int
	CompressLevel     int
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
func (c *Cache) Set(key string, object interface{}, timeout int) {
	b, err := c.Codec.Encode(&object)
	if err != nil {
		log.Errorf("Error encoding object for key %s: %s", key, err)
		return
	}
	c.SetBytes(key, b, timeout)
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
		log.Errorf("Error decoding object for key %s: %s", key, err)
	}
	return false
}

// GetMulti returns several objects as a map[string]interface{}
// in only one roundtrip to the cache
func (c *Cache) GetMulti(keys []string) map[string]interface{} {
	if c.manipulatesKeys() {
		k := make([]string, len(keys))
		for ii, v := range keys {
			k[ii] = c.backendKey(v)
		}
		keys = k
	}
	data, err := c.Backend.GetMulti(keys)
	if err != nil {
		log.Errorf("Error querying cache for keys %v: %v", keys, err)
		return nil
	}
	objects := make(map[string]interface{}, len(data))
	if c.manipulatesKeys() {
		for k, v := range data {
			var object interface{}
			err := c.Codec.Decode(v, &object)
			if err != nil {
				log.Errorf("Error decoding object for key %s: %s", k, err)
				continue
			}
			objects[c.frontendKey(k)] = object
		}
	} else {
		for k, v := range data {
			var object interface{}
			err := c.Codec.Decode(v, &object)
			if err != nil {
				log.Errorf("Error decoding object for key %s: %s", k, err)
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
func (c *Cache) SetBytes(key string, b []byte, timeout int) {
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
	err := c.Backend.Set(c.backendKey(key), b, timeout)
	if err != nil {
		log.Errorf("Error setting cache key %s: %s", key, err)
	}
}

// GetBytes returns the byte array assocciated with the given key
func (c *Cache) GetBytes(key string) []byte {
	b, err := c.Backend.Get(c.backendKey(key))
	if err != nil {
		log.Errorf("Error getting cache key %s: %s", key, err)
		return nil
	}
	if c.MinCompressLength >= 0 && len(b) > 0 {
		log.Debugf("Decompressing key %s", key)
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
	return b
}

func (c *Cache) Delete(key string) {
	err := c.Backend.Delete(c.backendKey(key))
	if err != nil {
		log.Errorf("Error deleting cache key %s: %s", key, err)
	}
}

func (c *Cache) Close() {
	c.Backend.Close()
}

func RegisterBackend(scheme string, f BackendInitializer) {
	backends[scheme] = f
}

func RegisterCodec(name string, codec *Codec) {
	codecs[name] = codec
}

func SetDefaultUrl(url string) {
	defaultCacheUrl = url
}

func DefaultUrl() string {
	return defaultCacheUrl
}

func New(cacheUrl string) *Cache {
	cache := &Cache{
		MinCompressLength: -1,
		CompressLevel:     zlib.DefaultCompression,
	}
	var query url.Values
	u, err := url.Parse(cacheUrl)
	if err != nil {
		if err != nil && cacheUrl != "" {
			log.Errorf("Invalid cache URL %q: %s\n", cacheUrl, err)
		}
	} else {
		query = u.Query()
	}
	var codec *Codec = nil
	if query != nil {
		codecName := query.Get("codec")
		if codecName != "" {
			if c, ok := codecs[codecName]; ok {
				codec = c
			} else {
				log.Errorf("Unknown cache codec name %q\n", codecName)
			}
		}
		cache.Prefix = query.Get("prefix")
		if mcl := query.Get("min_compress"); mcl != "" {
			val, err := strconv.Atoi(mcl)
			if err == nil {
				cache.MinCompressLength = val
			} else {
				log.Warningf("Invalid min_compress value %q (must be integer)", mcl)
			}
		}
		if cl := query.Get("compress_level"); cl != "" {
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
	}
	if codec == nil {
		codec = &GobEncoder
	}
	cache.Codec = codec
	var backendInitializer BackendInitializer
	if u != nil {
		backendInitializer = backends[u.Scheme]
		if backendInitializer == nil && cacheUrl != "" {
			log.Errorf("Unknown cache backend type %q\n", u.Scheme)
		}
	}
	if backendInitializer == nil {
		backendInitializer = InitializeDummyBackend
	}
	cache.Backend = backendInitializer(u)
	return cache
}

func NewDefault() *Cache {
	return New(defaultCacheUrl)
}
