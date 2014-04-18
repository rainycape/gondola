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

type Range interface {
	IsValid() bool
	Range() (*int64, *int64)
	Set(w http.ResponseWriter, total uint64)
	StatusCode() int
	String() string
}

type Server interface {
	Serve(w http.ResponseWriter, id string, rng Range) error
}

func Register(name string, o Opener) {
	registry[name] = o
}

func Get(name string) Opener {
	return registry[name]
}
