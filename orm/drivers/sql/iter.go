package sql

import (
	"database/sql"
	"gondola/orm/driver"
)

type Iter struct {
	model  driver.Model
	driver *Driver
	rows   *sql.Rows
	err    error
}

func (i *Iter) Next(out interface{}) bool {
	if i.err == nil && i.rows != nil && i.rows.Next() {
		var values []interface{}
		var transforms []*transform
		transforms, values, i.err = i.driver.outValues(i.model, out)
		if i.err == nil {
			i.err = i.rows.Scan(values...)
			for _, v := range transforms {
				i.err = i.driver.backend.TransformInValue(v.In, v.Out)
			}
		}
		return i.err == nil
	}
	return false
}

func (i *Iter) Err() error {
	return i.err
}

func NewIter(m driver.Model, r *sql.Rows, err error) driver.Iter {
	return &Iter{
		model: m,
		rows:  r,
		err:   err,
	}
}
