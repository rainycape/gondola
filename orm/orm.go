package orm

import (
	"fmt"
	"gondola/orm/driver"
	"gondola/orm/query"
	"reflect"
)

type Orm struct {
	driver driver.Driver
}

func (o *Orm) Query(m *Model, q query.Q) *Query {
	return &Query{
		orm:    o,
		model:  m,
		q:      q,
		limit:  -1,
		offset: -1,
	}
}

func (o *Orm) One(out interface{}, q query.Q) error {
	model, err := o.model(out)
	if err != nil {
		return err
	}
	iter := o.Query(model, q).Iter()
	if !iter.Next(out) {
		if err := iter.Err(); err != nil {
			return err
		}
		return ErrNotFound
	}
	return nil
}

func (o *Orm) Insert(obj interface{}) (Result, error) {
	model, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	return o.driver.Insert(model, obj)
}

func (o *Orm) MustInsert(obj interface{}) Result {
	res, err := o.Insert(obj)
	if err != nil {
		panic(err)
	}
	return res
}

func (o *Orm) Delete(m *Model, q query.Q) (Result, error) {
	return o.driver.Delete(m, q)
}

func (o *Orm) Close() error {
	if o.driver != nil {
		err := o.driver.Close()
		o.driver = nil
		return err
	}
	return nil
}

func (o *Orm) Driver() driver.Driver {
	return o.driver
}

func (o *Orm) model(obj interface{}) (*Model, error) {
	t := reflect.TypeOf(obj)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	model := typeRegistry[t]
	if model == nil {
		return nil, fmt.Errorf("no model registered for type %v", t)
	}
	return model, nil
}

func New(name string, params string) (*Orm, error) {
	opener := driver.Get(name)
	if opener == nil {
		return nil, fmt.Errorf("no ORM driver named %q", name)
	}
	drv, err := opener(params)
	if err != nil {
		return nil, fmt.Errorf("Error opening %q driver: %s", name, err)
	}
	return &Orm{
		driver: drv,
	}, nil
}
