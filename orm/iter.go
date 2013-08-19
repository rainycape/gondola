package orm

import (
	"gondola/orm/driver"
)

type Iter struct {
	q     *Query
	limit int
	driver.Iter
	err error
}

// Next advances the iter to the next result,
// filling the fields in the out parameter. It
// returns true iff there was a result.
func (i *Iter) Next(out interface{}) bool {
	if i.err != nil {
		return false
	}
	if i.Iter == nil {
		if i.q.model == nil {
			i.q.model, i.err = i.q.orm.model(out)
			if i.q.model == nil {
				return false
			}
		}
		i.q.orm.numQueries++
		i.Iter = i.q.orm.conn.Query(i.q.model, i.q.q, i.limit, i.q.offset, i.q.sortDir, i.q.sortField)
	}
	ok := i.Iter.Next(out)
	if ok {
		i.err = i.q.model.fields.Methods.Load(out)
	}
	return ok && i.err == nil
}

// Err returns the first error returned by the iterator. Once
// there's an error, Next() will return false.
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}
	if i.Iter != nil {
		return i.Iter.Err()
	}
	return nil
}

// P panics if the iter has an error. It's intended as a shorthand
// to save a few lines of code. For iterating over
// the results, a common pattern is:
//
//     for iter.Next(&obj) {
//     ... do something with obj...
//     }
//     if err := iter.Err(); err != nil {
//         panic(err)
//     }
//
// With P() you can instead use this code and save a few keystrokes:
//
//     for iter.Next(&obj) {
//     ... do something with obj...
//     }
//     iter.P()
func (i *Iter) P() {
	if err := i.Err(); err != nil {
		panic(err)
	}
}

// Close closes the iter. It's automatically called when the results
// are exhausted, but if you're ignoring some results you must call
// Close manually to avoid leaking resources. Close is idempotent.
func (i *Iter) Close() error {
	if i.Iter != nil {
		err := i.Iter.Close()
		i.Iter = nil
		return err
	}
	return nil
}
