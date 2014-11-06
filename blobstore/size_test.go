package blobstore

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "gnd.la/blobstore/driver/file"
	_ "gnd.la/blobstore/driver/leveldb"
	"gnd.la/config"
	"gnd.la/util/formatutil"
)

func totalSize(p string) int64 {
	var total int64
	filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func storeBinaries(t *testing.T, store *Blobstore, prepend []byte, count int) int {
	// Store all files in /usr/bin
	var stored int
	err := filepath.Walk("/usr/bin", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if count > 0 && stored >= count {
			return nil
		}
		if info.Mode().IsRegular() {
			stored++
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			w, err := store.Create()
			if err != nil {
				return err
			}
			if _, err := w.Write(prepend); err != nil {
				return err
			}
			if _, err := io.Copy(w, f); err != nil {
				return err
			}
			if err := w.Close(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	return stored
}

func testSize(t *testing.T, drv string) {
	count := 10
	if os.Getenv("BLOBSTORE_TEST_ALL") != "" {
		count = -1
	}
	if testing.Short() {
		t.SkipNow()
	}
	dir, err := ioutil.TempDir("", "pool-benchmark")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cfg := drv + "://" + dir
	u, err := config.ParseURL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(u)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	start1 := time.Now()
	stored := storeBinaries(t, store, nil, count)
	stored += storeBinaries(t, store, make([]byte, 32), count)
	t.Logf("took %s to store", time.Since(start1))
	t.Logf("%s size is %s", drv, formatutil.Size(uint64(totalSize(dir))))
	start2 := time.Now()
	// Verify that all files are OK
	iter, err := store.Iter()
	if err != nil {
		t.Fatal(err)
	}
	var id string
	found := 0
	for iter.Next(&id) {
		found++
		f, err := store.Open(id)
		if err != nil {
			t.Error(err)
			continue
		}
		if err := f.Check(); err != nil {
			t.Error(err)
		}
		f.Close()
	}
	if found != stored {
		t.Errorf("stored %d files, %d found", stored, found)
	}
	t.Logf("took %s to verify", time.Since(start2))
}

func TestFileSize(t *testing.T)    { testSize(t, "file") }
func TestLevelDBSize(t *testing.T) { testSize(t, "leveldb") }
