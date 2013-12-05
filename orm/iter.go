package orm

import (
	"gnd.la/orm/driver"
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
func (i *Iter) Next(out ...interface{}) bool {
	if i.err != nil {
		return false
	}
	if i.Iter == nil {
		if i.q.model == nil {
			i.q.model, i.q.methods, i.err = i.q.orm.models(out, i.q.q, i.q.jtype)
			if i.err != nil {
				return false
			}
		} else {
			i.q.methods = append(i.q.methods, i.q.model.fields.Methods)
			for cur := i.q.model.join; cur != nil; cur = cur.model.join {
				i.q.methods = append(i.q.methods, cur.model.fields.Methods)
			}
		}
		i.Iter = i.q.exec(i.limit)
	}
	ok := i.Iter.Next(out...)
	if ok {
		for ii, v := range out {
			if i.err = i.q.methods[ii].Load(v); i.err != nil {
				break
			}
		}
	} else {
		i.Close()
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
		ierr := i.Iter.Err()
		err := i.Iter.Close()
		i.Iter = nil
		if ierr != nil {
			i.err = ierr
		}
		return err
	}
	return nil
}
