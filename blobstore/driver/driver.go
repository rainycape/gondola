// Package driver includes the interfaces required to implement
// a blobstore driver.
package driver

import (
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

func Register(name string, o Opener) {
	registry[name] = o
}

func Get(name string) Opener {
	return registry[name]
}
