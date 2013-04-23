package assets

import (
	"gondola/loaders"
	"gondola/util"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

type Manager interface {
	Load(name string) (ReadSeekerCloser, time.Time, error)
	LoadURL(u *url.URL) (ReadSeekerCloser, time.Time, error)
	URL(name string) string
}

type AssetsManager struct {
	Loader       loaders.Loader
	prefix       string
	prefixLength int
}

func NewAssetsManager(loader loaders.Loader, prefix string) Manager {
	m := new(AssetsManager)
	m.Loader = loader
	m.prefix = prefix
	m.prefixLength = len(prefix)
	runtime.SetFinalizer(m, func(manager *AssetsManager) {
		manager.Close()
	})
	return m
}

func (m *AssetsManager) hash(name string) (string, error) {
	r, _, err := m.Load(name)
	if err != nil {
		return "", err
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return util.Adler32(b)[:6], nil
}

func (m *AssetsManager) Load(name string) (ReadSeekerCloser, time.Time, error) {
	return m.Loader.Load(name)
}

func (m *AssetsManager) LoadURL(u *url.URL) (ReadSeekerCloser, time.Time, error) {
	p := u.Path
	if !(p[1] == 'f' || p[1] == 'r') && !(p == "/favicon.ico" || p == "/robots.txt") {
		p = p[m.prefixLength:]
	}
	p = filepath.FromSlash(path.Clean("/" + p))
	return m.Load(p)
}

func (m *AssetsManager) URL(name string) string {
	suffix := ""
	h, _ := m.hash(name)
	if h != "" {
		suffix = "?v=" + h
	}
	return path.Clean(path.Join(m.prefix, name)) + suffix
}

func (m *AssetsManager) Close() error {
	return nil
}
