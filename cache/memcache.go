package cache

import (
	"github.com/bradfitz/gomemcache/memcache"
	"net/url"
	"strings"
)

type MemcacheBackend struct {
	*memcache.Client
}

func (c *MemcacheBackend) Set(key string, b []byte, timeout int) error {
	item := memcache.Item{Key: key, Value: b, Expiration: int32(timeout)}
	return c.Client.Set(&item)
}

func (c *MemcacheBackend) Get(key string) ([]byte, error) {
	item, err := c.Client.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	if item != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (c *MemcacheBackend) GetMulti(keys []string) (map[string][]byte, error) {
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

func (c *MemcacheBackend) Delete(key string) error {
	err := c.Client.Delete(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return err
	}
	return nil
}

func init() {
	RegisterBackend("memcache", func(cacheUrl *url.URL) Backend {
		hosts := strings.Split(cacheUrl.Host, ",")
		client := memcache.New(hosts...)
		return &MemcacheBackend{Client: client}
	})
}
