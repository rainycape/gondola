package orm

import (
	"gnd.la/orm/operation"
	"gnd.la/orm/query"
)

func (o *Orm) Operate(q query.Q, table *Table, op *operation.Operation) (Result, error) {
	return o.conn.Operate(table.model, q, op)
}

func (o *Orm) MustOperate(q query.Q, table *Table, op *operation.Operation) Result {
	res, err := o.Operate(q, table, op)
	if err != nil {
		panic(err)
	}
	return res
}
