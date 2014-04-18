package blobstore

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"

	"gnd.la/blobstore/driver"
	"gnd.la/config"
)

var (
	imports = map[string]string{
		"file":   "gnd.la/blobstore/driver/file",
		"gridfs": "gnd.la/blobstore/driver/gridfs",
		"s3":     "gnd.la/blobstore/driver/s3",
	}
)

// Store represents a connection to a blobstore. Use New()
// to initialize a Store and Store.Close to close it.
type Store struct {
	drv driver.Driver
	srv driver.Server
}

// New returns a new *Store using the given url as its configure
// the URL scheme represents the driver used and the rest of the
// values in the URL are driver dependent. Please, see the package
// documentation for the available drivers and each driver sub-package
// for driver-specific documentation.
func New(url *config.URL) (*Store, error) {
	if url == nil {
		return nil, fmt.Errorf("blobstore is not configured")
	}
	opener := driver.Get(url.Scheme)
	if opener == nil {
		if imp := imports[url.Scheme]; imp != "" {
			return nil, fmt.Errorf("please import %q to use the blobstore driver %q", imp, url.Scheme)
		}
		return nil, fmt.Errorf("unknown blobstore driver %q. Perhaps you forgot an import?", url.Scheme)
	}
	drv, err := opener(url)
	if err != nil {
		return nil, fmt.Errorf("error opening blobstore driver %q: %s", url.Scheme, err)
	}
	s := &Store{
		drv: drv,
	}
	if srv, ok := drv.(driver.Server); ok {
		s.srv = srv
	}
	return s, nil
}

// Create returns a new file for writing and sets its metadata
// to meta (which might be nil). Note that the file should be
// closed by calling WFile.Close.
func (s *Store) Create() (*WFile, error) {
	return s.CreateId(newId())
}

// CreateId works like Create, but uses the given id rather than generating
// a new one. If a file with the same id already exists, it's overwritten.
func (s *Store) CreateId(id string) (*WFile, error) {
	w, err := s.drv.Create(id)
	if err != nil {
		return nil, err
	}
	return &WFile{
		id:       id,
		file:     w,
		dataHash: newHash(),
		store:    s,
	}, nil
}

// Open opens the file with the given id for reading. Note that
// the file should be closed by calling RFile.Close after you're
// done with it.
func (s *Store) Open(id string) (*RFile, error) {
	f, err := s.drv.Open(id)
	if err != nil {
		return nil, err
	}
	return &RFile{id: id, file: f, store: s}, nil
}

// ReadAll is a shorthand for Open(f).ReadAll()
func (s *Store) ReadAll(id string) (data []byte, err error) {
	f, err := s.Open(id)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadAll()
}

// Store works like StoreId, but generates a new id for the file.
func (s *Store) Store(b []byte, meta interface{}) (string, error) {
	return s.StoreId(newId(), b, meta)
}

// StoreId is a shorthand for storing the given data in b and the metadata
// in meta with the given file id. If a file with the same id exists, it's
// overwritten.
func (s *Store) StoreId(id string, b []byte, meta interface{}) (string, error) {
	f, err := s.CreateId(id)
	if err != nil {
		return "", err
	}
	if err := f.SetMeta(meta); err != nil {
		return "", err
	}
	if _, err := f.Write(b); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Id(), nil
}

// Remove deletes the file with the given id.
func (s *Store) Remove(id string) error {
	s.drv.Remove(s.metaName(id))
	return s.drv.Remove(id)
}

// Driver returns the underlying driver
func (s *Store) Driver() driver.Driver {
	return s.drv
}

// Serve servers the given file by writing it to the given http.ResponseWriter.
// Some drivers might be able to serve the file directly from their backend. Otherwise,
// the file will be read from the blobstore and written to w. The rng parameter might be
// used for sending a partial response to the client.
func (s *Store) Serve(w http.ResponseWriter, id string, rng *Range) error {
	if s.srv != nil {
		return s.srv.Serve(w, id, rng)
	}
	f, err := s.Open(id)
	if err != nil {
		return err
	}
	defer f.Close()
	size, err := f.Size()
	if err != nil {
		return err
	}
	var r io.Reader = f
	if rng.IsValid() {
		if rng.Start != nil {
			var offset int64
			if *rng.Start < 0 {
				offset = int64(size) + *rng.Start
			} else {
				offset = *rng.Start
			}
			if _, err := f.Seek(offset, os.SEEK_SET); err != nil {
				return err
			}
		}
		if rng.End != nil {
			r = &io.LimitedReader{R: r, N: int64(rng.Size(size))}
		}
	}
	rng.Set(w, size)
	w.WriteHeader(rng.StatusCode())
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

// Close closes the connection to the blobstore.
func (s *Store) Close() error {
	return s.drv.Close()
}

func (s *Store) metaName(id string) string {
	return id + ".meta"
}

func isNil(v interface{}) bool {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		return val.IsNil()
	}
	return false
}
