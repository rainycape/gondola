package driver

import (
	"gondola/orm/query"
)

var registry = map[string]Opener{}

type Opener func(params string) (Driver, error)

type Driver interface {
	MakeModels(m []Model) error
	Query(m Model, q query.Q, limit int, offset int, sort int, sortField string) Iter
	Insert(m Model, data interface{}) (Result, error)
	Update(m Model, data interface{}, q query.Q) (Result, error)
	Upsert(m Model, data interface{}, q query.Q) (Result, error)
	Delete(m Model, q query.Q) (Result, error)
	Close() error
	// True if the driver can perform upserts
	Upserts() bool
	// List of struct tags to be read, in decreasing order of priority.
	// The first non-empty tag is used.
	Tags() []string
}

func Register(name string, opener Opener) {
	registry[name] = opener
}

func Get(name string) Opener {
	return registry[name]
}
