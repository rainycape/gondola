// Package driver includes the interfaces required to implement
// a blobstore driver.
package driver

import (
	"net/http"

	"gnd.la/config"
)

var (
	registry = map[string]Opener{}
)

type Opener func(url *config.URL) (Driver, error)

type Driver interface {
	Create(id string) (WFile, error)
	Open(id string) (RFile, error)
	Remove(id string) error
	Close() error
}

// Iter is the same interface as gnd.la/blobstore.Iter. See its
// documentation.
type Iter interface {
	Next(id *string) bool
	Err() error
	Close() error
}

// Iterable is the interface implemented by drivers which can iteratet
// over the files stored in them.
type Iterable interface {
	Iter() (Iter, error)
}

type Range interface {
	IsValid() bool
	Range() (*int64, *int64)
	Set(w http.ResponseWriter, total uint64)
	StatusCode() int
	String() string
}

type Server interface {
	// Serve serves the file directly from the driver to the given
	// http.ResponseWriter. If this function returns (false, nil)
	// the blobstore will request the file to the driver and serve
	// it by copying its contents to w. Any non nil error return
	// will cause the blobstore to return an error to the caller.
	Serve(w http.ResponseWriter, id string, rng Range) (bool, error)
}

func Register(name string, o Opener) {
	registry[name] = o
}

func Get(name string) Opener {
	return registry[name]
}
