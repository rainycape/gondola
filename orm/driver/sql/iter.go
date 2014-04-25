package sql

import (
	"database/sql"
	"gnd.la/orm/driver"
	"reflect"
)

type Iter struct {
	model  driver.Model
	driver *Driver
	rows   *sql.Rows
	err    error
}

func (i *Iter) Next(out ...interface{}) bool {
	if i.err == nil && i.rows != nil && i.rows.Next() {
		var vals []reflect.Value
		var fields []*driver.Fields
		var values []interface{}
		var scanners [][]*scanner
		model := i.model
		for model.Skip() {
			model = model.Join().Model()
		}
		for ii := range out {
			v := out[ii]
			if isNil(v) {
				continue
			}
			val, vfields, vvalues, vscanners, err := i.driver.outValues(model, v)
			if err != nil {
				i.err = err
				return false
			}
			vals = append(vals, val)
			fields = append(fields, vfields)
			values = append(values, vvalues...)
			scanners = append(scanners, vscanners)
			if j := model.Join(); j != nil {
				model = j.Model()
				for model.Skip() {
					join := model.Join()
					if join == nil {
						break
					}
					model = join.Model()
				}
			}
		}
		i.err = i.rows.Scan(values...)
		for ii, f := range fields {
			if f == nil {
				// This model was skipped
				continue
			}
			vscanners := scanners[ii]
			nilVal := true
			for _, p := range f.Pointers {
				isNil := true
				for jj, v := range f.Indexes {
					if f.IsSubfield(v, p) && !vscanners[jj].Nil {
						isNil = false
						nilVal = false
						break
					}
				}
				if isNil {
					fval := i.driver.fieldByIndex(vals[ii], p, false)
					fval.Set(reflect.Zero(fval.Type()))
				}
			}
			if nilVal {
				for _, v := range vscanners {
					if !v.Nil {
						nilVal = false
						break
					}
				}
			}
			if nilVal {
				val := reflect.ValueOf(out[ii]).Elem()
				val.Set(reflect.Zero(val.Type()))
			}
		}
		for _, s := range scanners {
			for _, v := range s {
				scannerPool.Put(v)
			}
		}
		return i.err == nil
	}
	return false
}

func (i *Iter) Err() error {
	return i.err
}

func (i *Iter) Close() error {
	if i.rows != nil {
		return i.rows.Close()
	}
	return nil
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
