package blobstore

import (
	"fmt"
	"hash/adler32"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "gnd.la/blobstore/driver/file"
	_ "gnd.la/blobstore/driver/gridfs"
	_ "gnd.la/blobstore/driver/leveldb"
	_ "gnd.la/blobstore/driver/s3"
	"gnd.la/config"
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

func randData(size int) []byte {
	b := make([]byte, size)
	for ii := 0; ii < len(b); ii += 50 {
		b[ii] = byte(rand.Int31n(256))
	}
	return b
}

var linearIndex = 0

func linearData(size int) []byte {
	b := make([]byte, size)
	for ii := range b {
		b[ii] = byte(linearIndex)
	}
	linearIndex++
	if linearIndex == 50 {
		linearIndex = 0
	}
	return b
}

func testStore(t *testing.T, meta *Meta, cfg string) {
	t.Logf("Testing store with config %s", cfg)
	u, err := config.ParseURL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(u)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var ids []string
	var hashes []uint32
	for ii := 0; ii < 10; ii++ {
		var r []byte
		if ii%2 == 0 {
			r = linearData(dataSize)
		} else {
			r = randData(dataSize)
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
		defer f.Close()
		t.Logf("Opened file %s", v)
		s, err := f.Size()
		if err != nil {
			t.Error(err)
		} else if s != dataSize {
			t.Errorf("Invalid data size for file %s. Want %v, got %v.", v, dataSize, s)
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
		if err := f.Check(); err != nil {
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
	return
	// Now remove all the files
	for _, v := range ids {
		if err := store.Remove(v); err != nil {
			t.Error(err)
		} else {
			t.Logf("deleted file %s", v)
		}
	}
	// Check that the files do not exist
	for _, v := range ids {
		if f, err := store.Open(v); err == nil || f != nil {
			t.Errorf("expecting nil file and non-nil err, got file %v and err %v instead", f, err)
			if f == nil {
				f.Close()
			}
		} else {
			t.Logf("file %s was deleted", v)
		}
	}
}

func TestFileStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "pool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cfg := "file://" + dir
	testStore(t, nil, cfg)
}

func TestFileStoreMeta(t *testing.T) {
	dir, err := ioutil.TempDir("", "pool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cfg := "file://" + dir
	testStore(t, &Meta{Foo: 5}, cfg)
}

func TestGridfs(t *testing.T) {
	if !testPort(27017) {
		t.Skip("mongodb is not running. start mongodb on localhost to run this test")
	}
	testStore(t, nil, "gridfs://localhost/blobstore_test")
}

func TestS3(t *testing.T) {
	b, err := ioutil.ReadFile("s3.txt")
	if err != nil || !strings.HasPrefix(string(b), "s3://") {
		abs, _ := filepath.Abs("s3.txt")
		t.Skipf("please, provide a file with an s3 blobstore url at %s to execute this test (e.g. \"s3://my-blobstore-test?access_key=akey&secret_key=some_secret\"", abs)
	}
	testStore(t, nil, strings.TrimSpace(string(b)))
}

func TestLevelDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "pool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cfg := "leveldb://" + dir
	testStore(t, &Meta{Foo: 5}, cfg)
}

const (
	modeR  = 1 << 0
	modeW  = 1 << 1
	modeRW = modeR | modeW

	opCount = 100
)

func benchmarkWrite(b *testing.B, bs *Blobstore, size int, f func(int) []byte) []string {
	ids := make([]string, 0, opCount)
	for ii := 0; ii < opCount; ii++ {
		id, err := bs.Store(f(size), nil)
		if err != nil {
			b.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func benchmarkRead(b *testing.B, bs *Blobstore, ids []string) {
	for _, v := range ids {
		f, err := bs.Open(v)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(ioutil.Discard, f); err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}

func benchmarkDriver(b *testing.B, drv string, size int, mode int, f func(int) []byte) {
	b.ReportAllocs()
	s := int64(opCount * size)
	b.SetBytes(s)
	dir, err := ioutil.TempDir("", "pool-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cfg := drv + "://" + dir
	u, err := config.ParseURL(cfg)
	if err != nil {
		b.Fatal(err)
	}
	store, err := New(u)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	b.ResetTimer()
	switch {
	case mode&modeR != 0 && mode&modeW != 0:
		b.SetBytes(s * 2)
		for ii := 0; ii < b.N; ii++ {
			ids := benchmarkWrite(b, store, size, f)
			benchmarkRead(b, store, ids)
		}
	case mode&modeW != 0:
		for ii := 0; ii < b.N; ii++ {
			benchmarkWrite(b, store, size, f)
		}
	case mode&modeR != 0:
		ids := benchmarkWrite(b, store, size, f)
		b.ResetTimer()
		for ii := 0; ii < b.N; ii++ {
			benchmarkRead(b, store, ids)
		}
	}
}
