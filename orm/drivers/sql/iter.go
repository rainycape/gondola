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
		var scanners []scanner
		values, scanners, i.err = i.driver.outValues(i.model, out)
		if i.err == nil {
			i.err = i.rows.Scan(values...)
		}
		for _, v := range scanners {
			v.Put()
		}
		return i.err == nil
	}
	return false
}

func (i *Iter) Err() error {
	return i.err
}

func NewIter(m driver.Model, d driver.Driver, r *sql.Rows, err error) driver.Iter {
	// TODO: Check for errors here?
	drv, _ := d.(*Driver)
	return &Iter{
		model:  m,
		driver: drv,
		rows:   r,
		err:    err,
	}
}
