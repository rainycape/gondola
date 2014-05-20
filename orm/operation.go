package orm

import (
	"errors"

	"gnd.la/orm/operation"
	"gnd.la/orm/query"
)

var (
	errNoOperations = errors.New("no operations specified")
)

func (o *Orm) Operate(table *Table, q query.Q, ops ...*operation.Operation) (Result, error) {
	if len(ops) == 0 {
		return nil, errNoOperations
	}
	return o.conn.Operate(table.model, q, ops)
}

func (o *Orm) MustOperate(table *Table, q query.Q, ops ...*operation.Operation) Result {
	res, err := o.Operate(table, q, ops...)
	if err != nil {
		panic(err)
	}
	return res
}
