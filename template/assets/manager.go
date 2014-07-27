package assets

import (
	"io"
	"net/url"
	"os"
	"path"
	"runtime"
	"sync"

	"gnd.la/crypto/hashutil"
	"gnd.la/net/urlutil"

	"gopkgs.com/vfs.v1"
)

type Manager struct {
	fs           vfs.VFS
	prefix       string
	prefixLength int
	cache        map[string]string
	mutex        sync.RWMutex
}

func New(fs vfs.VFS, prefix string) *Manager {
	m := new(Manager)
	m.cache = make(map[string]string)
	m.fs = fs
	m.SetPrefix(prefix)
	runtime.SetFinalizer(m, func(manager *Manager) {
		manager.Close()
	})
	return m
}

func (m *Manager) hash(name string) (string, error) {
	f, err := m.Load(name)
	if err != nil {
		return "", err
	}
	h := hashutil.Adler32(f)
	f.Close()
	return h[:6], nil
}

func (m *Manager) VFS() vfs.VFS {
	return m.fs
}

func (m *Manager) Path(u *url.URL) string {
	p := u.Path
	if !(p[1] == 'f' || p[1] == 'r') && !(p == "/favicon.ico" || p == "/robots.txt") {
		p = p[m.prefixLength:]
	}
	return path.Clean(p)
}

func (m *Manager) Load(name string) (io.ReadCloser, error) {
	return m.fs.Open(name)
}

func (m *Manager) LoadURL(u *url.URL) (io.ReadCloser, error) {
	return m.Load(m.Path(u))
}

func (m *Manager) Has(name string) bool {
	st, err := m.fs.Stat(name)
	return err == nil && st.Mode().IsRegular()
}

func (m *Manager) Create(name string, overwrite bool) (io.WriteCloser, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if !overwrite {
		flags |= os.O_EXCL
	}
	f, err := m.fs.OpenFile(name, flags, 0644)
	if err != nil && vfs.IsNotExist(err) {
		// Try to create the directory
		if err2 := vfs.MkdirAll(m.fs, path.Dir(name), 0755); err2 == nil {
			f, err = m.fs.OpenFile(name, flags, 0644)
		}
	}
	return f, err
}

func (m *Manager) URL(name string) string {
	if urlutil.IsURL(name) {
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
		return clean + "?v=" + h
	}
	return clean
}

func (m *Manager) Prefix() string {
	return m.prefix
}

func (m *Manager) SetPrefix(prefix string) {
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}
	m.prefix = prefix
	m.prefixLength = len(prefix)
}

func (m *Manager) Close() error {
	return nil
}
