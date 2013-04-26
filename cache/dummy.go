package cache

import (
	"net/url"
)

type DummyBackend struct {
}

func (c *DummyBackend) Set(key string, b []byte, timeout int) error {
	return nil
}

func (c *DummyBackend) Get(key string) ([]byte, error) {
	return nil, nil
}

func (c *DummyBackend) GetMulti(keys []string) (map[string][]byte, error) {
	return nil, nil
}

func (c *DummyBackend) Delete(key string) error {
	return nil
}

func (c *DummyBackend) Close() error {
	return nil
}

func (c *DummyBackend) Connection() interface{} {
	return nil
}

func InitializeDummyBackend(cacheUrl *url.URL) Backend {
	return &DummyBackend{}
}

func init() {
	RegisterBackend("dummy", InitializeDummyBackend)
}
