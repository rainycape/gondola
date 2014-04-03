package driver

import (
	"fmt"
	"gnd.la/config"
	"gnd.la/util/formatutil"
	"runtime"
	"sort"
	"sync"
	"time"
)

type item struct {
	data    []byte
	expires int64
}

type keyedItem struct {
	key  string
	item *item
}

type byExpirationAndSize []*keyedItem

func (e byExpirationAndSize) Len() int {
	return len(e)
}

func (e byExpirationAndSize) Less(i, j int) bool {
	ei := e[i].item.expires
	ej := e[j].item.expires
	if ei != 0 && ej != 0 && ei != ej {
		return ei < ej
	}
	if ei != 0 && ej == 0 {
		return true
	}
	if ei == 0 && ej != 0 {
		return false
	}
	return len(e[i].item.data) > len(e[j].item.data)
}

func (e byExpirationAndSize) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

var cache struct {
	sync.RWMutex
	items map[string]*item
	size  uint64
}

type MemoryDriver struct {
	maxSize uint64
	prune   chan struct{}
	mu      sync.Mutex
}

func (d *MemoryDriver) Set(key string, b []byte, timeout int) error {
	var expires int64
	if timeout != 0 {
		expires = time.Now().Unix() + int64(timeout)
	}
	prevSize := uint64(0)
	cache.Lock()
	if prev := cache.items[key]; prev != nil {
		prevSize = uint64(len(prev.data))
	}
	cache.items[key] = &item{
		data:    b,
		expires: expires,
	}
	cache.size += uint64(len(b)) - prevSize
	if d.maxSize > 0 && cache.size > d.maxSize {
		d.mu.Lock()
		// Unlock before sending over the channel,
		// otherwise we might cause a deadlock since
		// the pruneWorker might be waiting for the
		// cache lock to be released while the send
		// might be blocking waiting for the pruneWorker.
		cache.Unlock()
		d.prune <- struct{}{}
		d.mu.Unlock()
		return nil
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
		d.deleteItem(key, item)
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
	if d.prune != nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		close(d.prune)
		d.prune = nil
	}
	return nil
}

func (d *MemoryDriver) Connection() interface{} {
	return nil
}

func (d *MemoryDriver) pruneWorker(ch <-chan struct{}) {
	for _ = range ch {
		d.pruneCache()
	}
}

func (d *MemoryDriver) pruneCache() {
	cache.Lock()
	defer cache.Unlock()
	if cache.size < d.maxSize {
		return
	}
	items := make([]*keyedItem, 0, len(cache.items))
	for k, v := range cache.items {
		items = append(items, &keyedItem{
			key:  k,
			item: v,
		})
	}
	sort.Sort(byExpirationAndSize(items))
	threshold := uint64(float64(d.maxSize) * 0.9)
	for _, v := range items {
		delete(cache.items, v.key)
		cache.size -= uint64(len(v.item.data))
		if cache.size < threshold {
			break
		}
	}
}

func openMemoryDriver(value string, o config.Options) (Driver, error) {
	mdrv := &MemoryDriver{}
	if o != nil {
		if ms, ok := o["max_size"]; ok {
			maxSize, err := formatutil.ParseSize(ms)
			if err != nil {
				return nil, fmt.Errorf("invalid max_size %q", ms)
			}
			mdrv.maxSize = maxSize
			mdrv.prune = make(chan struct{}, runtime.GOMAXPROCS(0))
			go mdrv.pruneWorker(mdrv.prune)
		}
	}
	return mdrv, nil
}

func init() {
	cache.items = make(map[string]*item)
	Register("memory", openMemoryDriver)
}
