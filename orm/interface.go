package orm

import (
	"gnd.la/orm/query"
)

// Interface is implemented by both Orm
// and Transaction. This allows functions to
// receive an orm.Interface parameter and work
// with both transactions and outside of them.
// See the Orm documentation to find what each
// method does.
type Interface interface {
	Table(t *Table) *Query
	Exists(t *Table, q query.Q) (bool, error)
	Count(t *Table, q query.Q) (uint64, error)
	Query(q query.Q) *Query
	One(q query.Q, out ...interface{}) error
	All() *Query
	Insert(obj interface{}) (Result, error)
	MustInsert(obj interface{}) Result
	InsertInto(t *Table, obj interface{}) (Result, error)
	MustInsertInto(t *Table, obj interface{}) Result
	Update(q query.Q, obj interface{}) (Result, error)
	MustUpdate(q query.Q, obj interface{}) Result
	Upsert(q query.Q, obj interface{}) (Result, error)
	MustUpsert(q query.Q, obj interface{}) Result
	Save(obj interface{}) (Result, error)
	MustSave(obj interface{}) Result
	SaveInto(t *Table, obj interface{}) (Result, error)
	MustSaveInto(t *Table, obj interface{}) Result
	DeleteFromTable(t *Table, q query.Q) (Result, error)
	Delete(obj interface{}) error
	MustDelete(obj interface{})
	DeleteFrom(t *Table, obj interface{}) error
	Begin() (*Tx, error)
}
