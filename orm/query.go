package orm

import (
	"gondola/orm/query"
)

type Query struct {
	orm    *Orm
	model  *Model
	q      query.Q
	limit  int
	offset int
}

func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

func (q *Query) Offset(offset int) *Query {
	q.offset = offset
	return q
}

func (q *Query) Iter() Iter {
	return q.orm.driver.Query(q.model, q.q, q.limit, q.offset)
}
