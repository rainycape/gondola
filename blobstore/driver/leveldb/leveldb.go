package leveldb

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"gnd.la/blobstore/driver"
	"gnd.la/config"
	"gnd.la/util/pathutil"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	syncOptions = &opt.WriteOptions{Sync: true}
)

type leveldbDriver struct {
	files  *leveldb.DB
	chunks *leveldb.DB
	dir    string
}

func (d *leveldbDriver) Create(id string) (driver.WFile, error) {
	return newWFile(d, id), nil
}

func (d *leveldbDriver) Open(id string) (driver.RFile, error) {
	value, err := d.files.Get([]byte(id), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("file %s not found", id)
		}
		return nil, err
	}
	count := int(littleEndian.Uint32(value))
	if count == 0 {
		// Data is inline
		return &rfile{chunks: [][]byte{value[4:]}}, nil
	}
	pos := 4
	chunks := make([][]byte, count)
	for ii := 0; ii < count; ii++ {
		size := int(littleEndian.Uint32(value[pos:]))
		pos += 4
		key := value[pos : pos+size]
		chunk, err := d.chunks.Get(key, nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				return nil, fmt.Errorf("chunk %s in file %s not found", hex.EncodeToString(key), id)
			}
			return nil, err
		}
		chunks[ii] = chunk
		pos += size
	}
	return &rfile{chunks: chunks}, nil
}

func (d *leveldbDriver) Remove(id string) error {
	return d.files.Delete([]byte(id), syncOptions)
}

func (d *leveldbDriver) Close() error {
	if err := d.files.Close(); err != nil {
		return err
	}
	if err := d.chunks.Close(); err != nil {
		return err
	}
	return nil
}

func (d *leveldbDriver) Iter() (driver.Iter, error) {
	iter := d.files.NewIterator(nil, nil)
	return &leveldbIter{iter: iter}, nil
}

func leveldbOpener(url *config.URL) (driver.Driver, error) {
	value := url.Value
	if !filepath.IsAbs(value) {
		value = pathutil.Relative(value)
	}
	filesDir := filepath.Join(value, "files")
	files, err := leveldb.OpenFile(filesDir, nil)
	if err != nil {
		return nil, err
	}
	chunksDir := filepath.Join(value, "chunks")
	chunks, err := leveldb.OpenFile(chunksDir, nil)
	if err != nil {
		return nil, err
	}
	return &leveldbDriver{
		files:  files,
		chunks: chunks,
		dir:    value,
	}, nil
}

type leveldbIter struct {
	iter iterator.Iterator
}

func (i *leveldbIter) Next(id *string) bool {
	for i.iter.Next() {
		key := string(i.iter.Key())
		if strings.HasSuffix(key, ".meta") {
			continue
		}
		if id != nil {
			*id = key
		}
		return true
	}
	return false
}

func (i *leveldbIter) Err() error {
	return i.iter.Error()
}

func (i *leveldbIter) Close() error {
	i.iter.Release()
	return nil
}

func init() {
	driver.Register("leveldb", leveldbOpener)
}
