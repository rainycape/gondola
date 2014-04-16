// +build appengine

package memcache

import (
	"time"

	"appengine"
	"appengine/memcache"

	"gnd.la/cache/driver"
	"gnd.la/config"
)

type memcacheDriver struct {
	c appengine.Context
}

func (c *memcacheDriver) Set(key string, b []byte, timeout int) error {
	item := &memcache.Item{Key: key, Value: b, Expiration: time.Duration(timeout) * time.Second}
	return memcache.Add(c.c, item)
}

func (c *memcacheDriver) Get(key string) ([]byte, error) {
	item, err := memcache.Get(c.c, key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	if item != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (c *memcacheDriver) GetMulti(keys []string) (map[string][]byte, error) {
	results, err := memcache.GetMulti(c.c, keys)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	value := make(map[string][]byte, len(results))
	for k, v := range results {
		value[k] = v.Value
	}
	return value, nil
}

func (c *memcacheDriver) Delete(key string) error {
	err := memcache.Delete(c.c, key)
	if err != nil && err != memcache.ErrCacheMiss {
		return err
	}
	return nil
}

func (c *memcacheDriver) Connection() interface{} {
	return c
}

func (c *memcacheDriver) SetContext(ctx appengine.Context) {
	c.c = ctx
}

func (c *memcacheDriver) Close() error {
	return nil
}

func memcacheOpener(value string, o config.Options) (driver.Driver, error) {
	return &memcacheDriver{}, nil
}

func init() {
	driver.Register("memcache", memcacheOpener)
}
