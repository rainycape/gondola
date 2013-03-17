package cache

import (
	log "logging"
	"net/url"
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
	Prefix  string
	Backend Backend
	Codec   *Codec
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

func (c *Cache) Set(key string, object interface{}, timeout int) {
	b, err := c.Codec.Encode(&object)
	if err != nil {
		log.Errorf("Error encoding object for key %s: %s", key, err)
		return
	}
	c.SetBytes(key, b, timeout)
}

func (c *Cache) Get(key string) interface{} {
	b := c.GetBytes(key)
	if b != nil {
		var object interface{}
		err := c.Codec.Decode(b, &object)
		if err != nil {
			log.Errorf("Error decoding object for key %s: %s", key, err)
		}
		return object
	}
	return nil
}

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

func (c *Cache) SetBytes(key string, b []byte, timeout int) {
	log.Debugf("Setting key %s", c.backendKey(key))
	err := c.Backend.Set(c.backendKey(key), b, timeout)
	if err != nil {
		log.Errorf("Error setting cache key %s: %s", key, err)
	}
}

func (c *Cache) GetBytes(key string) []byte {
	b, err := c.Backend.Get(c.backendKey(key))
	if err != nil {
		log.Errorf("Error getting cache key %s: %s", key, err)
		return nil
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

func SetDefaultCacheUrl(url string) {
	defaultCacheUrl = url
}

func DefaultCacheUrl() string {
	return defaultCacheUrl
}

func New(cacheUrl string) *Cache {
	cache := &Cache{}
	var query url.Values
	u, err := url.Parse(cacheUrl)
	if err != nil {
		if err != nil {
			log.Errorf("Invalid cache URL '%s': %s\n", cacheUrl, err)
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
				log.Errorf("Unknown cache codec name '%s'\n", codecName)
			}
		}
		cache.Prefix = query.Get("prefix")
	}
	if codec == nil {
		codec = &GobEncoder
	}
	cache.Codec = codec
	var backendInitializer BackendInitializer
	if u != nil {
		backendInitializer = backends[u.Scheme]
		if backendInitializer == nil {
			log.Errorf("Unknown cache backend type '%s'\n", u.Scheme)
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
