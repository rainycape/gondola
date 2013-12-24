package orm

import (
	"gnd.la/orm/operation"
	"gnd.la/orm/query"
)

func (o *Orm) Operate(table *Table, q query.Q, op *operation.Operation) (Result, error) {
	return o.conn.Operate(table.model, q, op)
}

func (o *Orm) MustOperate(table *Table, q query.Q, op *operation.Operation) Result {
	res, err := o.Operate(table, q, op)
	if err != nil {
		panic(err)
	}
	return res
}
