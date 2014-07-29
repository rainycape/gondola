// +build appengine

package datastore

import (
	"fmt"
	"reflect"
	"strings"

	"gnd.la/config"
	"gnd.la/log"
	"gnd.la/orm/driver"
	"gnd.la/orm/operation"
	"gnd.la/orm/query"
	"gnd.la/util/types"

	"appengine"
	"appengine/datastore"
)

type Driver struct {
	c      appengine.Context
	logger *log.Logger
}

func (d *Driver) Check() error {
	return nil
}

func (d *Driver) Initialize(ms []driver.Model) error {
	// No need to create tables in the datastore. Instead,
	// check that the models can be stored.
	return nil
}

func (d *Driver) Query(m driver.Model, q query.Q, sort []driver.Sort, limit int, offset int) driver.Iter {
	dq, err := d.makeQuery(m, q, sort, limit, offset)
	if err != nil {
		return &Iter{err: err}
	}
	return &Iter{iter: dq.Run(d.c)}
}

func (d *Driver) Count(m driver.Model, q query.Q, limit int, offset int) (uint64, error) {
	dq, err := d.makeQuery(m, q, nil, limit, offset)
	if err != nil {
		return 0, err
	}
	c, err := dq.Count(d.c)
	return uint64(c), err
}

func (d *Driver) Exists(m driver.Model, q query.Q) (bool, error) {
	dq, err := d.makeQuery(m, q, nil, 1, -1)
	if err != nil {
		return false, err
	}
	c, err := dq.Count(d.c)
	return c != 0, err
}

func (d *Driver) Insert(m driver.Model, data interface{}) (driver.Result, error) {
	var id int64
	fields := m.Fields()
	var pkVal *reflect.Value
	// TODO: If the PK is supplied by the user rather than auto-assigned, it
	// might conflict with PKs generated by datastore.AllocateIDs().
	if fields.PrimaryKey >= 0 {
		p := d.primaryKey(fields, data)
		if p.IsValid() && types.Kind(p.Kind()) == types.Int {
			id = p.Int()
			if id == 0 {
				// Must assign PK field value after calling AllocateIDs
				pkVal = &p
			}
		}
	}
	name := m.Table()
	// Make all objects of a given kind ancestors of the same key. While
	// this hurts scalability, it makes all reads strongly consistent.
	parent := d.parentKey(m)
	var err error
	if id == 0 {
		id, _, err = datastore.AllocateIDs(d.c, name, parent, 1)
		if err != nil {
			return nil, err
		}
	}
	if fields.AutoincrementPk && pkVal != nil {
		pkVal.SetInt(int64(id))
	}
	key := datastore.NewKey(d.c, name, "", id, parent)
	log.Debugf("DATASTORE: put %s %v", key, data)
	_, err = datastore.Put(d.c, key, data)
	if err != nil {
		return nil, err
	}
	return &result{key: key, count: 1}, nil
}

func (d *Driver) Operate(m driver.Model, q query.Q, ops []*operation.Operation) (driver.Result, error) {
	return nil, fmt.Errorf("datastore driver does not support Operate")
}

func (d *Driver) Update(m driver.Model, q query.Q, data interface{}) (driver.Result, error) {
	keys, err := d.getKeys(m, q)
	if err != nil {
		return nil, err
	}
	src := make([]interface{}, len(keys))
	for ii := range src {
		src[ii] = data
	}
	// Multi variants need to be run in transactions, otherwise some
	// might fail while others succeed
	err = datastore.RunInTransaction(d.c, func(c appengine.Context) error {
		_, e := datastore.PutMulti(c, keys, src)
		return e
	}, nil)
	if err != nil {
		return nil, err
	}
	return &result{count: len(keys)}, nil
}

func (d *Driver) Upsert(m driver.Model, q query.Q, data interface{}) (driver.Result, error) {
	return nil, nil
}

func (d *Driver) Delete(m driver.Model, q query.Q) (driver.Result, error) {
	keys, err := d.getKeys(m, q)
	if err != nil {
		return nil, err
	}
	// See comment around PutMulti
	err = datastore.RunInTransaction(d.c, func(c appengine.Context) error {
		return datastore.DeleteMulti(c, keys)
	}, nil)
	if err != nil {
		return nil, err
	}
	return &result{count: len(keys)}, nil
}

func (d *Driver) getKeys(m driver.Model, q query.Q) ([]*datastore.Key, error) {
	dq, err := d.makeQuery(m, q, nil, -1, -1)
	if err != nil {
		return nil, err
	}
	iter := dq.KeysOnly().Run(d.c)
	var keys []*datastore.Key
	for {
		key, err := iter.Next(nil)
		if err != nil {
			if err == datastore.Done {
				break
			}
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (d *Driver) makeQuery(m driver.Model, q query.Q, sort []driver.Sort, limit int, offset int) (*datastore.Query, error) {
	if m.Join() != nil {
		return nil, errJoinNotSupported
	}
	dq := datastore.NewQuery(m.Table()).Ancestor(d.parentKey(m))
	var err error
	if dq, err = d.applyQuery(m, dq, q); err != nil {
		return nil, err
	}
	for _, v := range sort {
		field := v.Field()
		if v.Direction() == driver.DESC {
			field = "-" + field
		}
		dq = dq.Order(field)
	}
	if limit >= 0 {
		dq = dq.Limit(limit)
	}
	if offset > 0 {
		dq = dq.Offset(limit)
	}
	return dq, nil
}

func (d *Driver) parentKey(m driver.Model) *datastore.Key {
	return datastore.NewKey(d.c, m.Table(), "", -1, nil)
}

func (d *Driver) applyQuery(m driver.Model, dq *datastore.Query, q query.Q) (*datastore.Query, error) {
	var field *query.Field
	var op string
	switch x := q.(type) {
	case *query.Eq:
		field = &x.Field
		op = " ="
	case *query.Lt:
		field = &x.Field
		op = " <"
	case *query.Lte:
		field = &x.Field
		op = " <="
	case *query.Gt:
		field = &x.Field
		op = " >"
	case *query.Gte:
		field = &x.Field
		op = " >="
	case *query.And:
		var err error
		for _, v := range x.Conditions {
			dq, err = d.applyQuery(m, dq, v)
			if err != nil {
				return nil, err
			}
		}
	case nil:
	default:
		return nil, fmt.Errorf("datastore does not support %T queries", q)
	}
	if field != nil {
		if _, ok := field.Value.(query.F); ok {
			return nil, fmt.Errorf("datastore queries can't reference other properties (%v)", field.Value)
		}
		name := field.Field
		if strings.IndexByte(name, '.') >= 0 {
			// GAE flattens embedded fields, so we must remove
			// the parts of the field which refer to a flattened
			// field.
			fields := m.Fields()
			if idx, ok := fields.QNameMap[name]; ok {
				indexes := fields.Indexes[idx]
				parts := strings.Split(name, ".")
				if len(indexes) == len(parts) {
					var final []string
					typ := fields.Type
					for ii, v := range indexes {
						f := typ.Field(v)
						if !f.Anonymous {
							final = append(final, parts[ii])
						}
						typ = f.Type
					}
					name = strings.Join(final, ".")
				}
			}
		}
		log.Debugf("DATASTORE: filter %s %s %v", m, name+op, field.Value)
		dq = dq.Filter(name+op, field.Value)
	}
	return dq, nil
}

func (d *Driver) Close() error {
	return nil
}

func (d *Driver) Upserts() bool {
	return false
}

func (d *Driver) Tags() []string {
	return []string{"datastore"}
}

func (d *Driver) SetLogger(logger *log.Logger) {
	d.logger = logger
}

func (d *Driver) SetContext(ctx appengine.Context) {
	d.c = ctx
}

func (d *Driver) Begin() (driver.Tx, error) {
	return nil, errTransactionNotSupported
}

func (d *Driver) Commit() error {
	return driver.ErrNotInTransaction
}

func (d *Driver) Rollback() error {
	return driver.ErrNotInTransaction
}

func (d *Driver) Transaction(f func(driver.Driver) error) error {
	return datastore.RunInTransaction(d.c, func(c appengine.Context) error {
		drv := *d
		drv.c = c
		return f(&drv)
	}, nil)
}

func (d *Driver) Capabilities() driver.Capability {
	return driver.CAP_TRANSACTION | driver.CAP_AUTO_ID | driver.CAP_EVENTUAL | driver.CAP_PK
}

func (d *Driver) primaryKey(f *driver.Fields, data interface{}) reflect.Value {
	return fieldByIndex(reflect.ValueOf(data), f.Indexes[f.PrimaryKey])
}

func (d *Driver) HasFunc(fname string, retType reflect.Type) bool {
	return false
}

func (d *Driver) Connection() interface{} {
	return d.c
}

func fieldByIndex(val reflect.Value, indexes []int) reflect.Value {
	for _, v := range indexes {
		if val.Type().Kind() == reflect.Ptr {
			if val.IsNil() {
				return reflect.Value{}
			}
			val = val.Elem()
		}
		val = val.Field(v)
	}
	return val
}

func datastoreOpener(url *config.URL) (driver.Driver, error) {
	return &Driver{}, nil
}

func init() {
	driver.Register("datastore", datastoreOpener)
}
