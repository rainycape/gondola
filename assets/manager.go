package assets

import (
	"fmt"
	"gnd.la/loaders"
	"gnd.la/log"
	"gnd.la/util/hashutil"
	"io"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Manager interface {
	Load(name string) (loaders.ReadSeekCloser, time.Time, error)
	LoadURL(u *url.URL) (loaders.ReadSeekCloser, time.Time, error)
	Create(name string) (io.WriteCloser, error)
	URL(name string) string
	Debug() bool
	SetDebug(debug bool)
	Prefix() string
}

type assetsManager struct {
	watcher      *Watcher
	Loader       loaders.Loader
	debug        bool
	prefix       string
	prefixLength int
	cache        map[string]string
	mutex        sync.RWMutex
}

func NewManager(loader loaders.Loader, prefix string) Manager {
	m := new(assetsManager)
	m.cache = make(map[string]string)
	m.Loader = loader
	m.prefix = prefix
	m.prefixLength = len(prefix)
	runtime.SetFinalizer(m, func(manager *assetsManager) {
		manager.Close()
	})
	m.watch()
	return m
}

func (m *assetsManager) watch() {
	if dirloader, ok := m.Loader.(loaders.DirLoader); ok {
		watcher, err := NewWatcher(dirloader.Dir(), func(name string, deleted bool) {
			m.mutex.RLock()
			_, ok := m.cache[name]
			m.mutex.RUnlock()
			if ok {
				m.mutex.Lock()
				if deleted {
					delete(m.cache, name)
				} else {
					h, err := m.hash(name)
					if err == nil {
						m.cache[name] = h
					} else {
						delete(m.cache, name)
					}
				}
				m.mutex.Unlock()
			}
		})
		if err != nil {
			log.Warningf("Error creating watcher for %s: %s", dirloader.Dir, err)
		} else if watcher != nil {
			if err := watcher.Watch(); err == nil {
				m.watcher = watcher
			} else {
				log.Warningf("Error watching %s: %s", dirloader.Dir, err)
				watcher.Close()
			}
		}
	}
}

func (m *assetsManager) hash(name string) (string, error) {
	r, _, err := m.Load(name)
	if err != nil {
		return "", err
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return hashutil.Adler32(b)[:6], nil
}

func (m *assetsManager) Load(name string) (loaders.ReadSeekCloser, time.Time, error) {
	return m.Loader.Load(name)
}

func (m *assetsManager) LoadURL(u *url.URL) (loaders.ReadSeekCloser, time.Time, error) {
	p := u.Path
	if !(p[1] == 'f' || p[1] == 'r') && !(p == "/favicon.ico" || p == "/robots.txt") {
		p = p[m.prefixLength:]
	}
	p = filepath.FromSlash(path.Clean(p))
	return m.Load(p)
}

func (m *assetsManager) Create(name string) (io.WriteCloser, error) {
	return m.Loader.Create(name)
}

func (m *assetsManager) URL(name string) string {
	if strings.HasPrefix(name, "//") || strings.Contains(name, "://") {
		return name
	}
	m.mutex.RLock()
	h, ok := m.cache[name]
	m.mutex.RUnlock()
	if !ok {
		h, _ = m.hash(name)
		m.mutex.Lock()
		m.cache[name] = h
		m.mutex.Unlock()
	}
	clean := path.Clean(path.Join(m.prefix, name))
	if h != "" {
		return fmt.Sprintf("%s?v=%s", clean, h)
	}
	return clean
}

func (m *assetsManager) Debug() bool {
	return m.debug
}

func (m *assetsManager) SetDebug(debug bool) {
	m.debug = debug
}

func (m *assetsManager) Prefix() string {
	return m.prefix
}

func (m *assetsManager) Close() error {
	if m.watcher != nil {
		err := m.watcher.Close()
		m.watcher = nil
		return err
	}
	return nil
}
