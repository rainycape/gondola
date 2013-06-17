package orm

import (
	"fmt"
	"gondola/orm/query"
)

type Query struct {
	orm       *Orm
	model     *model
	q         query.Q
	limit     int
	offset    int
	sortField string
	sortDir   int
	err       error
}

// Table sets the table for the query. If the table was
// previously set, it's overridden. Rather than using
// strings to select tables, a Table object (which is
// returned from Register) is used. This way is not
// possible to mistype a table name, which avoids lots
// of errors.
func (q *Query) Table(t *Table) *Query {
	q.model = t.model
	return q
}

// Filter adds another condition to the query. In other
// words, it AND the previous condition with the one passed in.
func (q *Query) Filter(qu query.Q) *Query {
	if qu != nil {
		if q.q == nil {
			q.q = qu
		} else {
			switch x := q.q.(type) {
			case *query.And:
				x.Conditions = append(x.Conditions, qu)
			default:
				q.q = And(q.q, qu)
			}
		}
	}
	return q
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
	iter := q.iter(1)
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
func (q *Query) Iter() *Iter {
	return q.iter(q.limit)
}

// Count returns the number of results for the query. Note that
// you have to set the table manually before calling Count().
func (q *Query) Count() (uint64, error) {
	if q.model == nil {
		return 0, fmt.Errorf("no table selected, set one with Table() before calling Count()")
	}
	q.orm.numQueries++
	return q.orm.driver.Count(q.model, q.q, q.limit, q.offset, q.sortDir, q.sortField)
}

// MustCount works like Count, but panics if there's an error.
func (q *Query) MustCount() uint64 {
	c, err := q.Count()
	if err != nil {
		panic(err)
	}
	return c
}

// Clone returns a copy of the query.
func (q *Query) Clone() *Query {
	return &Query{
		orm:       q.orm,
		model:     q.model,
		q:         q.q,
		limit:     q.limit,
		offset:    q.offset,
		sortField: q.sortField,
		sortDir:   q.sortDir,
		err:       q.err,
	}
}

func (q *Query) iter(limit int) *Iter {
	return &Iter{
		q:     q,
		limit: limit,
		err:   q.err,
	}
}
