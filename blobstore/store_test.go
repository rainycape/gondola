package blobstore

import (
	"fmt"
	_ "gnd.la/blobstore/driver/file"
	_ "gnd.la/blobstore/driver/gridfs"
	"hash/adler32"
	"io/ioutil"
	"net"
	"os"
	"testing"
)

const (
	dataSize = 1 << 20 // 1MiB
)

func testPort(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

type Meta struct {
	Foo int
}

func fileData(t *testing.T, file string, size int64) []byte {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b := make([]byte, size)
	if _, err := f.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func randData(t *testing.T, size int64) []byte {
	return fileData(t, "/dev/urandom", size)
}

func zeroData(t *testing.T, size int64) []byte {
	return fileData(t, "/dev/zero", size)
}

func testStore(t *testing.T, meta *Meta, config string) {
	t.Logf("Testing store with config %s", config)
	store, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	var hashes []uint32
	for ii := 0; ii < 10; ii++ {
		var r []byte
		if ii%2 == 0 {
			r = zeroData(t, dataSize)
		} else {
			r = randData(t, dataSize)
		}
		id, err := store.Store(r, meta)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Logf("Stored file with id %s", id)
		ids = append(ids, id)
		hashes = append(hashes, adler32.Checksum(r))
	}
	for ii, v := range ids {
		f, err := store.Open(v)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Logf("Opened file %s", v)
		if f.Size() != dataSize {
			t.Errorf("Invalid data size for file %s. Want %v, got %v.", v, dataSize, f.Size())
		}
		if meta != nil {
			var m Meta
			if err := f.GetMeta(&m); err != nil {
				t.Errorf("error loading metadata from %v: %s", v, err)
			} else {
				if m.Foo != meta.Foo {
					t.Errorf("Invalid metadata value. Want %v, got %v.", meta.Foo, m.Foo)
				}
			}
		}
		if err := f.Verify(); err != nil {
			t.Errorf("error checking file %v: %s", v, err)
		}
		b, err := f.ReadAll()
		if err != nil {
			t.Error(err)
			continue
		}
		if len(b) != dataSize {
			t.Errorf("expecting %d bytes, got %d instead", dataSize, len(b))
			continue
		}
		if h := adler32.Checksum(b); h != hashes[ii] {
			t.Errorf("invalid hash %v for file %v, expected %v", h, v, hashes[ii])
		}
	}
}

func TestFileStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "pool-test")
	if err != nil {
		t.Fatal(err)
	}
	//defer os.RemoveAll(dir)
	cfg := "file://" + dir
	testStore(t, nil, cfg)
}

func TestFileStoreMeta(t *testing.T) {
	dir, err := ioutil.TempDir("", "pool-test")
	if err != nil {
		t.Fatal(err)
	}
	//defer os.RemoveAll(dir)
	cfg := "file://" + dir
	testStore(t, &Meta{Foo: 5}, cfg)
}

func TestGridfs(t *testing.T) {
	if !testPort(27017) {
		t.Skip("mongodb is not running. mongodb memcache on localhost to run this test")
	}
	testStore(t, nil, "gridfs://localhost/blobstore_test")
}
