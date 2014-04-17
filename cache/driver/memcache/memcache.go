// +build !appengine

package memcache

import (
	"github.com/rainycape/gomemcache/memcache"
	"gnd.la/cache/driver"
	"gnd.la/config"
	"strings"
	"time"
)

type memcacheDriver struct {
	*memcache.Client
}

func (c *memcacheDriver) Set(key string, b []byte, timeout int) error {
	item := memcache.Item{Key: key, Value: b, Expiration: int32(timeout)}
	return c.Client.Set(&item)
}

func (c *memcacheDriver) Get(key string) ([]byte, error) {
	item, err := c.Client.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	if item != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (c *memcacheDriver) GetMulti(keys []string) (map[string][]byte, error) {
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

func (c *memcacheDriver) Delete(key string) error {
	err := c.Client.Delete(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return err
	}
	return nil
}

func (c *memcacheDriver) Connection() interface{} {
	return c.Client
}

func memcacheOpener(url *config.URL) (driver.Driver, error) {
	hosts := strings.Split(url.Value, ",")
	conns := make([]string, len(hosts))
	for ii, v := range hosts {
		conns[ii] = driver.DefaultPort(v, 11211)
	}
	client := memcache.New(conns...)
	if tm, ok := url.Fragment.Int("timeout"); ok {
		client.SetTimeout(time.Millisecond * time.Duration(tm))
	}
	if maxIdle, ok := url.Fragment.Int("max_idle"); ok {
		client.SetMaxIdleConnsPerAddr(maxIdle)
	}
	return &memcacheDriver{Client: client}, nil
}

func init() {
	driver.Register("memcache", memcacheOpener)
}
