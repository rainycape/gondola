package sql

import (
	"database/sql"
	"fmt"
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
		var transforms []Transform
		transforms, values, i.err = i.driver.outValues(i.model, out)
		if i.err == nil {
			i.err = i.rows.Scan(values...)
			for _, v := range transforms {
				i.err = v.Transform()
			}
		}
		for ii, v := range values {
			switch x := v.(type) {
			case *int64:
				fmt.Printf("VAR %d INT64 %d\n", ii, *x)
			case *interface{}:
				fmt.Printf("VAR %d IFACE %v %T\n", ii, *x, *x)
			}
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
