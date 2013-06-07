package orm

import (
	"gondola/orm/query"
)

type Query struct {
	orm       *Orm
	model     *Model
	q         query.Q
	limit     int
	offset    int
	sortField string
	sortDir   int
}

// Limit sets the maximum number of results
// for the query.
func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

// Offset sets the offset for the query.
func (q *Query) Offset(offset int) *Query {
	q.offset = offset
	return q
}

// Sort sets the field and direction used for sorting
// this query.
func (q *Query) Sort(field string, dir SortDirection) *Query {
	q.sortField = field
	q.sortDir = int(dir)
	return q
}

// One fetches the first result for this query. If there
// are no results, it returns ErrNotFound.
func (q *Query) One(out interface{}) error {
	iter := q.orm.driver.Query(q.model, q.q, 1, q.offset)
	if iter.Next(out) {
		return nil
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return ErrNotFound
}

// Iter returns an Iter object which lets you
// iterate over the results produced by the
// query.
func (q *Query) Iter() Iter {
	return q.orm.driver.Query(q.model, q.q, q.limit, q.offset)
}
