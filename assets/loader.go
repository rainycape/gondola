package assets

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

type ReadSeekerCloser interface {
	io.ReadCloser
	io.Seeker
}

type Loader func(dir string, name string) (ReadSeekerCloser, time.Time, error)

func DefaultLoader(dir string, name string) (ReadSeekerCloser, time.Time, error) {
	p := filepath.FromSlash(path.Join(dir, name))
	f, err := os.Open(p)
	if err != nil {
		return nil, time.Time{}, err
	}
	s, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	return f, s.ModTime(), nil
}
