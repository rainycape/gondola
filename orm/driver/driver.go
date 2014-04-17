package driver

import (
	"gnd.la/config"
)

var (
	registry = map[string]Opener{}
)

type Opener func(url *config.URL) (Driver, error)

type Driver interface {
	Conn
	MakeTables(m []Model) error
	Begin() (Tx, error)
	Close() error
	// True if the driver can perform upserts
	Upserts() bool
	// List of struct tags to be read, in decreasing order of priority.
	// The first non-empty tag is used.
	Tags() []string
	Capabilities() Capability
}

func Register(name string, opener Opener) {
	registry[name] = opener
}

func Get(name string) Opener {
	return registry[name]
}
