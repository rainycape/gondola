package file

import (
	"gnd.la/blobstore/driver"
	"gnd.la/config"
	"gnd.la/util/pathutil"
	"os"
	"path/filepath"
)

const (
	idLength = 24
	sep      = idLength - 2
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
	return filepath.Join(f.dir, id[sep:], id[:sep])
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

func fsOpener(value string, o config.Options) (driver.Driver, error) {
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

func init() {
	driver.Register("file", fsOpener)
}
