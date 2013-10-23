package loaders

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

type mapWriter struct {
	bytes.Buffer
	name string
	m    map[string]*mapItem
	lock *sync.RWMutex
}

func (w *mapWriter) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.m[w.name] = &mapItem{
		data:  w.Bytes(),
		mtime: time.Now().UTC(),
	}
	return nil
}

func newMapWriter(name string, m map[string]*mapItem, lock *sync.RWMutex) *mapWriter {
	return &mapWriter{
		name: name,
		m:    m,
		lock: lock,
	}
}

type mapItem struct {
	data  []byte
	mtime time.Time
}

type mapLoader struct {
	sync.RWMutex
	items map[string]*mapItem
}

func (m *mapLoader) Load(name string) (ReadSeekCloser, time.Time, error) {
	m.RLock()
	item := m.items[name]
	m.RUnlock()
	if item == nil {
		return nil, time.Time{}, fmt.Errorf("resource %q does not exist", name)
	}
	return newReader(item.data), item.mtime, nil
}

func (m *mapLoader) Create(name string) (io.WriteCloser, error) {
	m.Lock()
	if _, ok := m.items[name]; ok {
		m.Unlock()
		return nil, fmt.Errorf("resource %q already exists", name)
	}
	if m.items == nil {
		m.items = make(map[string]*mapItem)
	}
	m.items[name] = nil
	m.Unlock()
	return newMapWriter(name, m.items, &m.RWMutex), nil
}

// MapLoader returns a Loader which loads resources from the
// given map.
func MapLoader(m map[string][]byte) Loader {
	items := map[string]*mapItem{}
	for k, v := range m {
		items[k] = &mapItem{
			data:  v,
			mtime: time.Now().UTC(),
		}
	}
	return &mapLoader{items: items}
}
