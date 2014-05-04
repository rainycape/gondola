package loaders

import (
	"time"
)

type memLoader struct {
	mapLoader
	loader Loader
}

func (m *memLoader) Load(name string) (ReadSeekCloser, time.Time, error) {
	f, t, err := m.mapLoader.Load(name)
	if err == nil {
		return f, t, nil
	}
	return m.loader.Load(name)
}

func (m *memLoader) List() ([]string, error) {
	loaderFiles, err := m.loader.List()
	if err != nil {
		return nil, err
	}
	memFiles, err := m.mapLoader.List()
	if err != nil {
		return nil, err
	}
	return append(loaderFiles, memFiles...), nil
}

// MemLoader wraps another loader, causing
// the created files to be stored in memory.
// It's usually used with FSLoader, to avoid
// creating temporary files.
func MemLoader(loader Loader) Loader {
	return &memLoader{mapLoader{}, loader}
}
