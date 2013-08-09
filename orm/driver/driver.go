package driver

import (
	"gondola/orm/query"
	"gondola/orm/transaction"
)

var registry = map[string]Opener{}

type Opener func(params string) (Driver, error)

type Driver interface {
	MakeTables(m []Model) error
	Query(m Model, q query.Q, limit int, offset int, sort int, sortField string) Iter
	Count(m Model, q query.Q, limit int, offset int, sort int, sortField string) (uint64, error)
	Exists(m Model, q query.Q) (bool, error)
	Insert(m Model, data interface{}) (Result, error)
	Update(m Model, q query.Q, data interface{}) (Result, error)
	Upsert(m Model, q query.Q, data interface{}) (Result, error)
	Delete(m Model, q query.Q) (Result, error)
	// Transactions
	Begin(transaction.Options) error
	Commit() error
	Rollback() error
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
