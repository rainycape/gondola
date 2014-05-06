package file

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"gnd.la/blobstore/driver"
	"gnd.la/config"
	"gnd.la/util/pathutil"
)

type fsDriver struct {
	dir    string
	tmpDir string
}

func (f *fsDriver) tmp(id string) string {
	return filepath.Join(f.tmpDir, id)
}

func (f *fsDriver) path(id string) string {
	// Use the last two bytes as the dirname, since the
	// two first increase monotonically with time
	ext := path.Ext(id)
	if ext != "" {
		id = id[:len(id)-len(ext)]
	}
	sep := len(id) - 2
	return filepath.Join(f.dir, id[sep:], id[:sep]+ext)
}

func (f *fsDriver) Create(id string) (driver.WFile, error) {
	tmp := filepath.Join(f.tmpDir, id)
	fp, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	return &wfile{
		File: fp,
		path: f.path(id),
	}, nil
}

func (f *fsDriver) Open(id string) (driver.RFile, error) {
	return os.Open(f.path(id))
}

func (f *fsDriver) Remove(id string) error {
	return os.Remove(f.path(id))
}

func (f *fsDriver) Close() error {
	return nil
}

func (f *fsDriver) Iter() (driver.Iter, error) {
	res, err := ioutil.ReadDir(f.dir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, v := range res {
		if v.IsDir() {
			name := v.Name()
			if name != "tmp" || name[0] != '.' {
				dirs = append(dirs, filepath.Join(f.dir, name))
			}
		}
	}
	return &fsIter{dirs: dirs}, nil
}

func fsOpener(url *config.URL) (driver.Driver, error) {
	value := url.Value
	if !filepath.IsAbs(value) {
		value = pathutil.Relative(value)
	}
	tmpDir := filepath.Join(value, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, err
	}
	return &fsDriver{
		dir:    value,
		tmpDir: tmpDir,
	}, nil
}

type fsIter struct {
	dirs     []string
	base     string
	dirIndex int
	names    []string
	err      error
}

func (f *fsIter) Next(id *string) bool {
	if id == nil {
		var discard string
		id = &discard
	}
	for len(f.names) == 0 {
		if f.dirIndex >= len(f.dirs) {
			*id = ""
			return false
		}
		cur := f.dirs[f.dirIndex]
		dir, err := os.Open(cur)
		if err != nil {
			f.err = err
			return false
		}
		names, err := dir.Readdirnames(-1)
		dir.Close()
		if err != nil {
			f.err = err
			return false
		}
		for _, v := range names {
			if filepath.Ext(v) != ".meta" {
				f.names = append(f.names, v)
			}
		}
		f.base = filepath.Base(cur)
		f.dirIndex++
	}
	*id = f.names[0] + f.base
	f.names = f.names[1:]
	return true
}

func (f *fsIter) Err() error {
	return f.err
}

func (f *fsIter) Close() error {
	return nil
}

func init() {
	driver.Register("file", fsOpener)
}
