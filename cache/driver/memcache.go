package driver

import (
	"github.com/bradfitz/gomemcache/memcache"
	"strings"
)

type MemcacheDriver struct {
	*memcache.Client
}

func (c *MemcacheDriver) Set(key string, b []byte, timeout int) error {
	item := memcache.Item{Key: key, Value: b, Expiration: int32(timeout)}
	return c.Client.Set(&item)
}

func (c *MemcacheDriver) Get(key string) ([]byte, error) {
	item, err := c.Client.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	if item != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (c *MemcacheDriver) GetMulti(keys []string) (map[string][]byte, error) {
	results, err := c.Client.GetMulti(keys)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	value := make(map[string][]byte, len(results))
	for k, v := range results {
		value[k] = v.Value
	}
	return value, nil
}

func (c *MemcacheDriver) Delete(key string) error {
	err := c.Client.Delete(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return err
	}
	return nil
}

func (c *MemcacheDriver) Close() error {
	return nil
}

func (c *MemcacheDriver) Connection() interface{} {
	return c.Client
}

func init() {
	Register("memcache", func(value string, o Options) Driver {
		hosts := strings.Split(value, ",")
		conns := make([]string, len(hosts))
		for ii, v := range hosts {
			conns[ii] = DefaultPort(v, 11211)
		}
		client := memcache.New(conns...)
		return &MemcacheDriver{Client: client}
	})
}
