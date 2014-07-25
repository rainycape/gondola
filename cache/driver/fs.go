package driver

import (
	"encoding/binary"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gnd.la/config"
	"gnd.la/crypto/hashutil"
	"gnd.la/util/pathutil"
)

type FileSystemDriver struct {
	Root string
}

func (f *FileSystemDriver) keyPath(key string) string {
	fileKey := hashutil.Md5(key)
	return filepath.Join(f.Root, fileKey[:2], fileKey[2:4], fileKey[4:])
}

func (f *FileSystemDriver) Set(key string, b []byte, timeout int) error {
	p := f.keyPath(key)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil {
		return err
	}
	fd, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	expiration := int64(timeout)
	if expiration > 0 {
		expiration += time.Now().Unix()
	}
	binary.Write(fd, binary.LittleEndian, expiration)
	total := len(b)
	for t := 0; t < total; {
		n, err := fd.Write(b)
		if err != nil {
			f.Delete(key)
			return err
		}
		t += n
	}
	return nil
}

func (f *FileSystemDriver) Get(key string) ([]byte, error) {
	fd, err := os.Open(f.keyPath(key))
	if err != nil {
		/* Cache miss */
		return nil, nil
	}
	defer fd.Close()
	var expiration int64
	binary.Read(fd, binary.LittleEndian, &expiration)
	if expiration > 0 && expiration < time.Now().Unix() {
		f.Delete(key)
		return nil, nil
	}
	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *FileSystemDriver) GetMulti(keys []string) (map[string][]byte, error) {
	value := make(map[string][]byte, len(keys))
	for _, k := range keys {
		result, err := f.Get(k)
		if err == nil {
			value[k] = result
		}
	}
	return value, nil
}

func (f *FileSystemDriver) Delete(key string) error {
	err := os.Remove(f.keyPath(key))
	return err
}

func (f *FileSystemDriver) Close() error {
	return nil
}

func (f *FileSystemDriver) Connection() interface{} {
	return nil
}

func (f *FileSystemDriver) Flush() error {
	return ErrNotImplemented
}

func fsOpener(url *config.URL) (Driver, error) {
	value := filepath.FromSlash(url.Value)
	if !filepath.IsAbs(value) {
		value = pathutil.Relative(value)
	}
	return &FileSystemDriver{Root: value}, nil
}

func init() {
	Register("file", fsOpener)
}
