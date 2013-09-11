package driver

import (
	"sync"
	"time"
)

type item struct {
	data    []byte
	expires int64
}

var mdrv *MemoryDriver

var cache struct {
	sync.RWMutex
	items map[string]*item
	size  uint64
}

type MemoryDriver struct {
}

func (d *MemoryDriver) Set(key string, b []byte, timeout int) error {
	var expires int64
	if timeout != 0 {
		expires = time.Now().Unix() + int64(timeout)
	}
	cache.Lock()
	cache.items[key] = &item{
		data:    b,
		expires: expires,
	}
	cache.Unlock()
	return nil
}

func (d *MemoryDriver) Get(key string) ([]byte, error) {
	cache.RLock()
	item := cache.items[key]
	cache.RUnlock()
	if item == nil {
		return nil, nil
	}
	if item.expires != 0 && item.expires < time.Now().Unix() {
		d.Delete(key)
		return nil, nil
	}
	return item.data, nil
}

func (d *MemoryDriver) GetMulti(keys []string) (map[string][]byte, error) {
	items := make(map[string]*item, len(keys))
	cache.RLock()
	for _, v := range keys {
		items[v] = cache.items[v]
	}
	cache.RUnlock()
	results := make(map[string][]byte, len(keys))
	now := time.Now().Unix()
	for k, v := range items {
		if v != nil {
			if v.expires != 0 && v.expires < now {
				d.deleteItem(k, v)
				continue
			}
			results[k] = v.data
		}
	}
	return results, nil
}

func (d *MemoryDriver) Delete(key string) error {
	cache.RLock()
	item := cache.items[key]
	cache.RUnlock()
	if item == nil {
		return nil
	}
	d.deleteItem(key, item)
	return nil
}

func (d *MemoryDriver) deleteItem(key string, i *item) {
	cache.Lock()
	delete(cache.items, key)
	cache.size -= uint64(len(i.data))
	cache.Unlock()
}

func (d *MemoryDriver) Close() error {
	cache.Lock()
	cache.items = make(map[string]*item)
	cache.size = 0
	cache.Unlock()
	return nil
}

func (d *MemoryDriver) Connection() interface{} {
	return nil
}

func openMemoryDriver(value string, o Options) (Driver, error) {
	// Don't do this in init(), since the memory driver
	// won't be used most of the time and we don't want
	// the user paying for these two allocations if they're
	// not going to use them.
	cache.Lock()
	if mdrv == nil {
		mdrv = &MemoryDriver{}
		cache.items = make(map[string]*item)
	}
	defer cache.Unlock()
	return mdrv, nil
}

func init() {
	Register("memory", openMemoryDriver)
}
