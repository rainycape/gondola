package loaders

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

type fsloader struct {
	dir string
}

func (f *fsloader) Load(name string) (ReadSeekCloser, time.Time, error) {
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

func (f *fsloader) Create(name string, overwrite bool) (io.WriteCloser, error) {
	p := filepath.FromSlash(path.Join(f.dir, name))
	flags := os.O_WRONLY | os.O_CREATE
	if !overwrite {
		flags |= os.O_EXCL
	}
	return os.OpenFile(p, flags, 0644)
}

func (f *fsloader) List() ([]string, error) {
	var names []string
	err := filepath.Walk(f.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			names = append(names, path[len(f.dir)+1:])
		}
		return nil
	})
	return names, err
}

func FSLoader(dir string) Loader {
	return &fsloader{dir: dir}
}
