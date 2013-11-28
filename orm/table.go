package orm

import (
	"gnd.la/orm/query"
)

type Table struct {
	model *joinModel
}

func (t *Table) Join(table *Table, q query.Q, jt JoinType) (*Table, error) {
	join := t.model.clone()
	if _, err := join.joinWith(table.model.model, q, jt); err != nil {
		return nil, err
	}
	return &Table{model: join}, nil
}

func (t *Table) MustJoin(table *Table, q query.Q, jt JoinType) *Table {
	tbl, err := t.Join(table, q, jt)
	if err != nil {
		panic(err)
	}
	return tbl
}

func (t *Table) Skip() *Table {
	model := t.model.clone()
	for cur := model; ; {
		cur.skip = true
		if cur.join == nil {
			break
		}
		cur = cur.join.model
	}
	model.skip = true
	return &Table{model: model}
}
