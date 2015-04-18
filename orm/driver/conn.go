package driver

import (
	"gnd.la/orm/operation"
	"gnd.la/orm/query"
)

type QueryOptions struct {
	Limit    int
	Offset   int
	Sort     []Sort
	Distinct bool
}

type Conn interface {
	Query(m Model, q query.Q, opts QueryOptions) Iter
	Count(m Model, q query.Q, opts QueryOptions) (uint64, error)
	Exists(m Model, q query.Q) (bool, error)
	Insert(m Model, data interface{}) (Result, error)
	Operate(m Model, q query.Q, ops []*operation.Operation) (Result, error)
	Update(m Model, q query.Q, data interface{}) (Result, error)
	Upsert(m Model, q query.Q, data interface{}) (Result, error)
	Delete(m Model, q query.Q) (Result, error)
	Connection() interface{}
}
