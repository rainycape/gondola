package driver

import (
	"reflect"

	"gnd.la/config"
)

var (
	registry = map[string]Opener{}
)

type Opener func(url *config.URL) (Driver, error)

type Driver interface {
	Conn
	Initialize(m []Model) error
	Begin() (Tx, error)
	Transaction(f func(Driver) error) error
	Close() error
	// True if the driver can perform upserts
	Upserts() bool
	// List of struct tags to be read, in decreasing order of priority.
	// The first non-empty tag is used.
	Tags() []string
	Capabilities() Capability
	HasFunc(fname string, retType reflect.Type) bool
}

func Register(name string, opener Opener) {
	registry[name] = opener
}

func Get(name string) Opener {
	return registry[name]
}
