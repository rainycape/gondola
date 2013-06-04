package sql

import (
	"database/sql"
	"gondola/orm/driver"
)

type Iter struct {
	model driver.Model
	rows  *sql.Rows
	err   error
}

func (i *Iter) Next(out interface{}) bool {
	if i.err == nil && i.rows != nil && i.rows.Next() {
		var values []interface{}
		values, i.err = i.model.Values(out)
		if i.err == nil {
			i.err = i.rows.Scan(values...)
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
