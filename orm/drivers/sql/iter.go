package sql

import (
	"database/sql"
	"gondola/orm/driver"
	"reflect"
)

type Iter struct {
	model  driver.Model
	driver *Driver
	rows   *sql.Rows
	err    error
}

func (i *Iter) Next(out interface{}) bool {
	if i.err == nil && i.rows != nil && i.rows.Next() {
		var val reflect.Value
		var fields *driver.Fields
		var values []interface{}
		var scanners []scanner
		val, fields, values, scanners, i.err = i.driver.outValues(i.model, out)
		if i.err == nil {
			i.err = i.rows.Scan(values...)
		}
		for _, p := range fields.Pointers {
			isNil := true
			for ii, v := range fields.Indexes {
				if fields.IsSubfield(v, p) && !scanners[ii].IsNil() {
					isNil = false
					break
				}
			}
			if isNil {
				fval := i.driver.fieldByIndex(val, p, false)
				fval.Set(reflect.Zero(fval.Type()))
			}
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

func (i *Iter) Close() error {
	return i.rows.Close()
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
