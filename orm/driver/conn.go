package driver

import (
	"gondola/orm/query"
)

type Conn interface {
	Query(m Model, q query.Q, limit int, offset int, sort int, sortField string) Iter
	Count(m Model, q query.Q, limit int, offset int) (uint64, error)
	Exists(m Model, q query.Q) (bool, error)
	Insert(m Model, data interface{}) (Result, error)
	Update(m Model, q query.Q, data interface{}) (Result, error)
	Upsert(m Model, q query.Q, data interface{}) (Result, error)
	Delete(m Model, q query.Q) (Result, error)
}
