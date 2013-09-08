package loaders

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

type writer struct {
	*bytes.Buffer
	name   string
	loader *mapLoader
}

func (w *writer) Close() error {
	w.loader.Lock()
	w.loader.items[w.name] = &item{
		data:  w.Bytes(),
		mtime: time.Now().UTC(),
	}
	w.loader.Unlock()
	return nil
}

type item struct {
	data  []byte
	mtime time.Time
}

type mapLoader struct {
	sync.RWMutex
	items map[string]*item
}

func (m *mapLoader) Load(name string) (ReadSeekCloser, time.Time, error) {
	m.RLock()
	item := m.items[name]
	m.RUnlock()
	if m == nil {
		return nil, time.Time{}, fmt.Errorf("resource %q does not exist", name)
	}
	return &reader{bytes.NewReader(item.data)}, item.mtime, nil
}

func (m *mapLoader) Create(name string) (io.WriteCloser, error) {
	m.Lock()
	if _, ok := m.items[name]; ok {
		m.Unlock()
		return nil, fmt.Errorf("resource %q already exists", name)
	}
	m.items[name] = nil
	m.Unlock()
	return &writer{bytes.NewBuffer(nil), name, m}, nil
}

// MapLoader returns a Loader which loads resources from the
// given map.
func MapLoader(m map[string][]byte) Loader {
	items := map[string]*item{}
	for k, v := range m {
		items[k] = &item{
			data:  v,
			mtime: time.Now().UTC(),
		}
	}
	return &mapLoader{items: items}
}
