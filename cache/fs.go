package cache

import (
	"encoding/binary"
	"gondola/util"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"time"
)

type FileSystemBackend struct {
	Root string
}

func (f *FileSystemBackend) keyPath(key string) string {
	fileKey := util.Md5([]byte(key))
	return path.Join(f.Root, fileKey[:2], fileKey[2:4], fileKey[4:])
}

func (f *FileSystemBackend) Set(key string, b []byte, timeout int) error {
	p := f.keyPath(key)
	err := os.MkdirAll(path.Dir(p), 0755)
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

func (f *FileSystemBackend) Get(key string) ([]byte, error) {
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

func (f *FileSystemBackend) GetMulti(keys []string) (map[string][]byte, error) {
	value := make(map[string][]byte, len(keys))
	for _, k := range keys {
		result, err := f.Get(k)
		if err == nil {
			value[k] = result
		}
	}
	return value, nil
}

func (f *FileSystemBackend) Delete(key string) error {
	err := os.Remove(f.keyPath(key))
	return err
}

func (f *FileSystemBackend) Close() error {
	return nil
}

func init() {
	RegisterBackend("file", func(cacheUrl *url.URL) Backend {
		var root string
		if cacheUrl.Host == "" {
			/* Absolute path */
			root = path.Join("/", cacheUrl.Path)
		} else {
			root = util.RelativePath(cacheUrl.Host)
		}
		return &FileSystemBackend{Root: root}
	})
}
