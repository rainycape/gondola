package orm

import (
	"fmt"
	"gnd.la/app/debug"
	"gnd.la/orm/driver"
	"gnd.la/orm/query"
	"reflect"
)

type Query struct {
	orm     *Orm
	model   *joinModel
	methods []*driver.Methods
	jtype   JoinType
	q       query.Q
	sort    []driver.Sort
	limit   int
	offset  int
	err     error
}

func (q *Query) ensureTable(f string) error {
	if q.model == nil {
		fmt.Errorf("no table selected, set one with Table() before calling %s()", f)
	}
	return nil
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

// Join sets the default join type for this query. If not
// specifed, an INNER JOIN is performed. Note that not all
// drivers support RIGHT joins (e.g. sqlite).
func (q *Query) Join(jt JoinType) *Query {
	q.jtype = jt
	return q
}

// Filter adds another condition to the query. In other
// words, it ANDs the previous condition with the one passed in.
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
// this query. To Sort by multiple fields, call Sort
// multiple times.
func (q *Query) Sort(field string, dir Sort) *Query {
	q.sort = append(q.sort, &querySort{
		field: field,
		dir:   driver.SortDirection(dir),
	})
	return q
}

// One fetches the first result for this query. If there
// are no results, it returns ErrNotFound.
func (q *Query) One(out ...interface{}) error {
	iter := q.iter(1)
	if iter.Next(out...) {
		// Must close the iter manually, because we're not
		// reaching the end.
		iter.Close()
		return nil
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return ErrNotFound
}

// Exists returns wheter a result with the specified query
// exists.
func (q *Query) Exists() (bool, error) {
	if err := q.ensureTable("Exists"); err != nil {
		return false, err
	}
	q.orm.numQueries++
	return q.orm.driver.Exists(q.model, q.q)
}

// Iter returns an Iter object which lets you
// iterate over the results produced by the
// query.
func (q *Query) Iter() *Iter {
	return q.iter(q.limit)
}

// All returns all results for this query in slices. Arguments
// to All must be pointers to slices of elements. Basically, All
// is a shortcut for:
//
//  var objs []*MyObject
//  var obj *MyObject
//  iter := some_query.Iter()
//  for iter.Next(obj) {
//	objs = append(objs, obj)
//  }
//  err := iter.Err()
//
// Using All instead, we can do the same in this shorter way:
//
//  var objs []*MyObject
//  err := some_query.All(&objects)
//
// Please, keep in mind that All will load all the objects into
// memory at the same time, so you shouldn't use it for large
// result sets.
func (q *Query) All(out ...interface{}) error {
	values := make([]reflect.Value, len(out))
	result := make([]interface{}, len(out))
	for ii, v := range out {
		val := reflect.ValueOf(v)
		if val.Kind() != reflect.Ptr {
			return fmt.Errorf("arguments to All() must be pointers to slices, argument %d is %T", ii+1, v)
		}
		elem := val.Type().Elem()
		if elem.Kind() != reflect.Slice {
			return fmt.Errorf("arguments to All() must be pointers to slices, argument %d is %T", ii+1, v)
		}
		result[ii] = reflect.New(elem.Elem()).Interface()
		values[ii] = val.Elem()
	}
	iter := q.Iter()
	for iter.Next(result...) {
		for ii, v := range values {
			v.Set(reflect.Append(v, reflect.ValueOf(result[ii]).Elem()))
		}
	}
	return iter.Err()
}

// MustAll works like All, but panics if there's an error.
func (q *Query) MustAll(out ...interface{}) {
	err := q.All(out...)
	if err != nil {
		panic(err)
	}
}

// Count returns the number of results for the query. Note that
// you have to set the table manually before calling Count().
func (q *Query) Count() (uint64, error) {
	if err := q.ensureTable("Count"); err != nil {
		return 0, err
	}
	q.orm.numQueries++
	return q.orm.driver.Count(q.model, q.q, q.limit, q.offset)
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
		orm:    q.orm,
		model:  q.model,
		q:      q.q,
		sort:   q.sort,
		limit:  q.limit,
		offset: q.offset,
		err:    q.err,
	}
}

func (q *Query) iter(limit int) *Iter {
	return &Iter{
		q:     q,
		limit: limit,
		err:   q.err,
	}
}

func (q *Query) exec(limit int) driver.Iter {
	if debug.On {
		defer debug.Startf(orm, "query").End()
	}
	q.orm.numQueries++
	return q.orm.conn.Query(q.model, q.q, q.sort, limit, q.offset)
}

// Field is a conveniency function which returns a reference to a field
// to be used in a query, mostly used for joins.
func F(field string) query.F {
	return query.F(field)
}

type querySort struct {
	field string
	dir   driver.SortDirection
}

func (s *querySort) Field() string {
	return s.field
}

func (s *querySort) Direction() driver.SortDirection {
	return s.dir
}
