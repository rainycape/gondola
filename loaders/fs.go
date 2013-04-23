package loaders

import (
	"os"
	"path"
	"path/filepath"
	"time"
)

type FSLoader struct {
	Dir string
}

func (f *FSLoader) Load(name string) (ReadSeekerCloser, time.Time, error) {
	p := filepath.FromSlash(path.Join(f.Dir, name))
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
	return &FSLoader{Dir: dir}
}
