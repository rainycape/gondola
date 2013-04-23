package loaders

import (
	"os"
	"path"
	"path/filepath"
	"time"
)

type FSLoader interface {
	Loader
	Dir() string
}

type fsloader struct {
	dir string
}

func (f *fsloader) Load(name string) (ReadSeekerCloser, time.Time, error) {
	p := filepath.FromSlash(path.Join(f.dir, name))
	fd, err := os.Open(p)
	if err != nil {
		return nil, time.Time{}, err
	}
	s, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, time.Time{}, err
	}
	return fd, s.ModTime(), nil
}

func NewFSLoader(dir string) Loader {
	return &fsloader{dir: dir}
}
