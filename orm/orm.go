package orm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gnd.la/app/profile"
	"gnd.la/config"
	"gnd.la/log"
	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/orm/query"
	"gnd.la/util/types"
)

var (
	imports = map[string]string{
		"postgres": "gnd.la/orm/driver/postgres",
		"sqlite":   "gnd.la/orm/driver/sqlite",
		"sqlite3":  "gnd.la/orm/driver/sqlite",
		"mysql":    "gnd.la/orm/driver/mysql",
	}
	errUntypedNilPointer = errors.New("untyped nil pointer passed to Next(). Please, cast it to the appropriate type e.g. (*MyType)(nil)")
	errNoModel           = errors.New("query without model - did you forget output parameters?")
	// Rollback is returned from functions passed to Orm.Transaction to
	// indicate that they want the transaction to be rolled back without
	// returning any error from Orm.Transaction.
	Rollback = errors.New("transaction rolled back")
)

const (
	// WILL_INITIALIZE is emitted just before a gnd.la/orm.Orm is
	// initialized. The object is a *gnd.la/orm.Orm.
	WILL_INITIALIZE = "gnd.la/orm.will-initialize"
	orm             = "orm"
)

type Orm struct {
	conn         driver.Conn
	driver       driver.Driver
	logger       *log.Logger
	tags         string
	typeRegistry typeRegistry
	// these fields are non-nil iff the ORM driver uses database/sql
	db *sql.DB
}

// Table returns a Query object initialized with the given table.
// The Table object is returned when registering the model. If you
// need to obtain a Table from a model type or name, see Orm.TypeTable
// and orm.NamedTable.
func (o *Orm) Table(t *Table) *Query {
	return &Query{
		orm:    o,
		model:  t.model,
		limit:  -1,
		offset: -1,
	}
}

// Exists is a shorthand for Table(t).Filter(q).Exists()
func (o *Orm) Exists(t *Table, q query.Q) (bool, error) {
	return o.Table(t).Filter(q).Exists()
}

// Count is a shorthand for Table(t).Filter(q).Count()
// Pass nil to count all the objects in the given table.
func (o *Orm) Count(t *Table, q query.Q) (uint64, error) {
	return o.Table(t).Filter(q).Count()
}

// Query returns a Query object, on which you can call
// Limit, Offset or Iter, to start iterating the results.
// If you want to iterate over all the items on a given table
// pass nil as the q argument.
// By default, the query will use the table of the first
// object passed to Next(), but you can override it using
// Table() (and you most do so for Count() and other functions
// which don't take objects).
func (o *Orm) Query(q query.Q) *Query {
	return &Query{
		orm:    o,
		q:      q,
		limit:  -1,
		offset: -1,
	}
}

// One is a shorthand for Query(q).One(&out)
func (o *Orm) One(q query.Q, out ...interface{}) (bool, error) {
	return o.Query(q).One(out...)
}

// MustOne works like One, but panics if there's an error.
func (o *Orm) MustOne(q query.Q, out ...interface{}) bool {
	ok, err := o.One(q, out...)
	if err != nil {
		panic(err)
	}
	return ok
}

// All is a shorthand for Query(nil)
func (o *Orm) All() *Query {
	return o.Query(nil)
}

// Insert saves an object into its collection. Its
// type must be previously registered as a model. If the model
// has an integer primary key with auto_increment, it will be
// be populated with the database assigned id.
func (o *Orm) Insert(obj interface{}) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	if err := m.fields.Methods.Save(obj); err != nil {
		return nil, err
	}
	return o.insert(m, obj)
}

// MustInsert works like Insert, but panics if there's
// an error.
func (o *Orm) MustInsert(obj interface{}) Result {
	res, err := o.Insert(obj)
	if err != nil {
		panic(err)
	}
	return res
}

// InsertInto works like insert, but stores the object in the
// given table (as returned by Register).
func (o *Orm) InsertInto(t *Table, obj interface{}) (Result, error) {
	if err := t.model.fields.Methods.Save(obj); err != nil {
		return nil, err
	}
	return o.insert(t.model.model, obj)
}

// MustInsertInto works like InsertInto, but panics if there's
// an error.
func (o *Orm) MustInsertInto(t *Table, obj interface{}) Result {
	res, err := o.InsertInto(t, obj)
	if err != nil {
		panic(err)
	}
	return res
}

func (o *Orm) insert(m *model, obj interface{}) (Result, error) {
	if profile.On {
		defer profile.Startf(orm, "insert").End()
	}
	var pkName string
	var pkVal reflect.Value
	f := m.fields
	if f.AutoincrementPk {
		pkName, pkVal = o.primaryKey(f, obj)
		if pkVal.Int() == 0 && !pkVal.CanSet() {
			typ := reflect.TypeOf(obj)
			return nil, fmt.Errorf("can't set primary key field %q. Please, insert a %v rather than a %v", pkName, reflect.PtrTo(typ), typ)
		}
	}
	if f.Defaults != nil {
		val := reflect.ValueOf(obj)
		for k, v := range f.Defaults {
			indexes := f.Indexes[k]
			fval := o.fieldByIndexCreating(val, indexes)
			isTrue, _ := types.IsTrueVal(fval)
			if !isTrue {
				if !fval.CanSet() {
					// Need to copy to alter the fields
					pval := reflect.New(val.Type())
					pval.Elem().Set(val)
					obj = pval.Interface()
					val = pval
					fval = o.fieldByIndexCreating(val, indexes)
				}
				if v.Kind() == reflect.Func {
					out := v.Call(nil)
					fval.Set(out[0])
				} else {
					fval.Set(v)
				}
			}
		}
	}
	res, err := o.conn.Insert(m, obj)
	if err == nil && pkVal.IsValid() && pkVal.Int() == 0 {
		id, err := res.LastInsertId()
		if err == nil && id != 0 {
			if o.logger != nil {
				o.logger.Debugf("Setting primary key %q to %d on model %v", pkName, id, m.Type())
			}
			pkVal.SetInt(id)
		} else if err != nil && o.logger != nil {
			o.logger.Errorf("could not obtain last insert id: %s", err)
		}
	}
	return res, err
}

func (o *Orm) Update(q query.Q, obj interface{}) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	if err := m.fields.Methods.Save(obj); err != nil {
		return nil, err
	}
	return o.update(m, q, obj)
}

// MustUpdate works like update, but panics if there's
// an error.
func (o *Orm) MustUpdate(q query.Q, obj interface{}) Result {
	res, err := o.Update(q, obj)
	if err != nil {
		panic(err)
	}
	return res
}

func (o *Orm) update(m *model, q query.Q, obj interface{}) (Result, error) {
	if profile.On {
		defer profile.Startf(orm, "update").End()
	}
	return o.conn.Update(m, q, obj)
}

// Upsert tries to perform an update with the given query
// and object. If there are not affected rows, it performs
// an insert. Some drivers (like mongodb) are able to perform
// this operation in just one query, but most require two
// trips to the database.
func (o *Orm) Upsert(q query.Q, obj interface{}) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	if err := m.fields.Methods.Save(obj); err != nil {
		return nil, err
	}
	if o.driver.Upserts() {
		if profile.On {
			defer profile.Startf(orm, "upsert").End()
		}
		return o.conn.Upsert(m, q, obj)
	}
	res, err := o.update(m, q, obj)
	if err != nil {
		return nil, err
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if aff == 0 {
		res, err = o.insert(m, obj)
	}
	return res, err
}

// MustUpsert works like Upsert, but panics if there's an error.
func (o *Orm) MustUpsert(q query.Q, obj interface{}) Result {
	res, err := o.Upsert(q, obj)
	if err != nil {
		panic(err)
	}
	return res
}

// Save takes an object, with its type registered as
// a model and either inserts it
// (if the primary key is zero or it has no primary key)
// or updates it using the primary key as the query
// (if it's non zero). If the update results in no
// affected rows, an insert will be performed. Save also
// supports models with composite keys. If any field forming
// the composite key is non-zero, an update will be tried
// before performing an insert.
func (o *Orm) Save(obj interface{}) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	if err := m.fields.Methods.Save(obj); err != nil {
		return nil, err
	}
	return o.save(m, obj)
}

// MustSave works like save, but panics if there's an
// error.
func (o *Orm) MustSave(obj interface{}) Result {
	res, err := o.Save(obj)
	if err != nil {
		panic(err)
	}
	return res
}

// SaveInto works like Save, but stores the object in the given
// table (as returned from Register).
func (o *Orm) SaveInto(t *Table, obj interface{}) (Result, error) {
	if err := t.model.fields.Methods.Save(obj); err != nil {
		return nil, err
	}
	return o.save(t.model.model, obj)
}

// MustSaveInto works like SaveInto, but panics if there's an error.
func (o *Orm) MustSaveInto(t *Table, obj interface{}) Result {
	res, err := o.SaveInto(t, obj)
	if err != nil {
		panic(err)
	}
	return res
}

func (o *Orm) save(m *model, obj interface{}) (Result, error) {
	if profile.On {
		defer profile.Startf(orm, "save").End()
	}
	var res Result
	var err error
	if m.fields.PrimaryKey >= 0 {
		pkName, pkVal := o.primaryKey(m.fields, obj)
		if driver.IsZero(pkVal) {
			return o.insert(m, obj)
		}
		res, err = o.update(m, Eq(pkName, pkVal.Interface()), obj)
	} else if len(m.fields.CompositePrimaryKey) > 0 {
		// Composite primary key
		names, values := o.compositePrimaryKey(m.fields, obj)
		for _, v := range values {
			if !driver.IsZero(v) {
				// We have a non-zero value, try to update
				qs := make([]query.Q, len(names))
				for ii := range names {
					qs[ii] = Eq(names[ii], values[ii].Interface())
				}
				res, err = o.update(m, And(qs...), obj)
				break
			}
		}
		if res == nil && err == nil {
			// Not updated. All the fields in the PK are zero
			return o.insert(m, obj)
		}
	} else {
		// No pk
		return o.insert(m, obj)
	}
	if err != nil {
		return nil, err
	}
	up, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if up == 0 {
		return o.insert(m, obj)
	}
	return res, nil
}

// DeleteFromTable removes all objects from the given table matching
// the query.
func (o *Orm) DeleteFromTable(t *Table, q query.Q) (Result, error) {
	return o.delete(t.model.model, q)
}

// Delete removes the given object, which must be of a type
// previously registered as a table and must have a primary key,
// either simple or composite.
func (o *Orm) Delete(obj interface{}) error {
	m, err := o.model(obj)
	if err != nil {
		return err
	}
	return o.deleteByPk(m, obj)
}

// MustDelete works like Delete, but panics if there's an error.
func (o *Orm) MustDelete(obj interface{}) {
	if err := o.Delete(obj); err != nil {
		panic(err)
	}
}

// DeleteFrom works like Delete, but deletes from the given table
// (as returned by Register)
func (o *Orm) DeleteFrom(t *Table, obj interface{}) error {
	return o.deleteByPk(t.model.model, obj)
}

func (o *Orm) deleteByPk(m *model, obj interface{}) error {
	var q query.Q
	if m.fields.PrimaryKey >= 0 {
		pkName, pkVal := o.primaryKey(m.fields, obj)
		if pkVal.IsValid() && pkName != "" {
			q = Eq(pkName, pkVal.Interface())
		}
	} else if len(m.fields.CompositePrimaryKey) > 0 {
		names, values := o.compositePrimaryKey(m.fields, obj)
		conditions := make([]query.Q, len(names))
		for ii, v := range names {
			conditions[ii] = Eq(v, values[ii].Interface())
		}
		q = And(conditions...)
	}
	if q == nil {
		return fmt.Errorf("type %T does not have a primary key", obj)
	}
	_, err := o.delete(m, q)
	return err
}

func (o *Orm) delete(m *model, q query.Q) (Result, error) {
	if profile.On {
		defer profile.Startf(orm, "delete").End()
	}
	return o.conn.Delete(m, q)
}

// Begin starts a new transaction. If the driver does
// not support transactions, Begin will return a fake
// transaction.
func (o *Orm) Begin() (*Tx, error) {
	caps := o.driver.Capabilities()
	if caps&driver.CAP_BEGIN == 0 {
		if caps&driver.CAP_TRANSACTION == 0 {
			return nil, fmt.Errorf("ORM driver %T does not support transactions", o.driver)
		}
		return nil, fmt.Errorf("ORM driver %T does not support Begin/Commit/Rollback - use Orm.Transaction instead", o.driver)
	}
	tx, err := o.driver.Begin()
	if err != nil {
		return nil, err
	}
	if o.logger != nil {
		o.logger.Debugf("Beginning transaction")
	}
	return &Tx{
		Orm: &Orm{
			conn:   tx,
			driver: o.driver,
			logger: o.logger,
			tags:   o.tags,
			db:     o.db,
		},
		o:  o,
		tx: tx,
	}, nil
}

// MustBegin works like Begin, but panics if there's an error.
func (o *Orm) MustBegin() *Tx {
	tx, err := o.Begin()
	if err != nil {
		panic(err)
	}
	return tx
}

// Transaction runs the given function f inside a transaction. This
// interface is provided because some drivers (most of the NoSQL based)
// don't support the Begin/Commit interface and must run a transaction
// as a function. To return from f rolling back the transaction without
// generating an error return in Transaction, use Rollback. Returning
// any other error will cause the transaction to be rolled back and the
// error will be returned from Transaction. If no errors are returned
// from f, the transaction is commited and the only error that might be
// returned from Transaction will be one produced while committing.
func (o *Orm) Transaction(f func(o *Orm) error) error {
	caps := o.driver.Capabilities()
	if caps&driver.CAP_TRANSACTION == 0 {
		return fmt.Errorf("ORM driver %T does not support transactions", o.driver)
	}
	if caps&driver.CAP_BEGIN != 0 {
		tx, err := o.Begin()
		if err != nil {
			return err
		}
		defer tx.Close()
		if err := f(tx.Orm); err != nil {
			if err == Rollback {
				err = tx.Rollback()
			}
			return err
		}
		return tx.Commit()
	}
	err := o.driver.Transaction(func(d driver.Driver) error {
		oc := *o
		oc.conn = d
		return f(&oc)
	})
	if err == Rollback {
		err = nil
	}
	return err
}

// Close closes the database connection. Since the ORM
// is thread safe and does its own connection pooling
// you should tipycally never call this function. Instead,
// create a ORM instance when starting up your application
// and always use it.
func (o *Orm) Close() error {
	if o.driver != nil {
		err := o.driver.Close()
		o.driver = nil
		return err
	}
	return nil
}

// Driver returns the underlying driver.
func (o *Orm) Driver() driver.Driver {
	return o.driver
}

// SqlDB returns the underlying database connection iff the
// ORM driver is using database/sql. Otherwise, it
// returns nil. Note that the returned value isn't of type
// database/sql.DB, but gnd.la/orm/driver/sql.DB, which is
// a small compatibility wrapper around the former. See the
// gnd.la/orm/driver/sql.DB documentation for further
// information.
func (o *Orm) SqlDB() *sql.DB {
	return o.db
}

// Logger returns the logger for this ORM. By default, it's
// nil.
func (o *Orm) Logger() *log.Logger {
	return o.logger
}

// SetLogger sets the logger for this ORM. If the underlying
// driver implements the interface orm.Logger, the logger will
// be set for it too (the sql driver implements this interface).
func (o *Orm) SetLogger(logger *log.Logger) {
	o.logger = logger
	if drvLogger, ok := o.driver.(Logger); ok {
		drvLogger.SetLogger(logger)
	}
}

func (o *Orm) models(objs []interface{}, q query.Q, sort []driver.Sort, jt JoinType) (*joinModel, []*driver.Methods, error) {
	jm := &joinModel{}
	models := make(map[*model]struct{})
	var methods []*driver.Methods
	for _, v := range objs {
		vm, err := o.model(v)
		if err != nil {
			return nil, nil, err
		}
		last, err := jm.joinWith(vm, nil, jt)
		if err != nil {
			return nil, nil, err
		}
		r := reflect.ValueOf(v)
		if r.Type().Kind() == reflect.Ptr && r.IsNil() {
			last.skip = true
		}
		models[vm] = struct{}{}
		methods = append(methods, vm.fields.Methods)
	}
	if jm.model == nil {
		return nil, nil, errNoModel
	}
	if q != nil {
		if err := jm.joinWithQuery(q, jt, models, &methods); err != nil {
			return nil, nil, err
		}
	}
	if sort != nil {
		if err := jm.joinWithSort(sort, jt, models, &methods); err != nil {
			return nil, nil, err
		}
	}
	return jm, methods, nil
}

func (o *Orm) fieldByIndex(val reflect.Value, indexes []int) reflect.Value {
	for _, v := range indexes {
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return reflect.Value{}
			}
			val = val.Elem()
		}
		val = val.Field(v)
	}
	return val
}

func (o *Orm) fieldByIndexCreating(val reflect.Value, indexes []int) reflect.Value {
	for _, v := range indexes {
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				val.Set(reflect.New(val.Type().Elem()))
			}
			val = val.Elem()
		}
		val = val.Field(v)
	}
	return val
}

func (o *Orm) primaryKey(f *driver.Fields, obj interface{}) (string, reflect.Value) {
	pk := f.PrimaryKey
	if pk < 0 {
		return "", reflect.Value{}
	}
	val := driver.Direct(reflect.ValueOf(obj))
	return f.QNames[pk], o.fieldByIndex(val, f.Indexes[pk])
}

func (o *Orm) compositePrimaryKey(f *driver.Fields, obj interface{}) ([]string, []reflect.Value) {
	if len(f.CompositePrimaryKey) == 0 {
		return nil, nil
	}
	val := driver.Direct(reflect.ValueOf(obj))
	var names []string
	var values []reflect.Value
	for _, v := range f.CompositePrimaryKey {
		names = append(names, f.QNames[v])
		values = append(values, o.fieldByIndex(val, f.Indexes[v]))
	}
	return names, values
}

// Open creates a new ORM using the specified
// configuration URL.
func New(url *config.URL) (*Orm, error) {
	name := url.Scheme
	opener := driver.Get(name)
	if opener == nil {
		if imp, ok := imports[name]; ok {
			return nil, fmt.Errorf("please, import package %q to use driver %q", imp, name)
		}
		return nil, fmt.Errorf("no ORM driver named %q", name)
	}
	drv, err := opener(url)
	if err != nil {
		return nil, fmt.Errorf("error opening ORM driver %q: %s", name, err)
	}
	if err := drv.Check(); err != nil {
		return nil, err
	}
	tags := strings.Join(drv.Tags(), "-")
	globalRegistry.RLock()
	typeRegistry := globalRegistry.types[tags].clone()
	globalRegistry.RUnlock()
	o := &Orm{
		conn:         drv,
		driver:       drv,
		tags:         tags,
		typeRegistry: typeRegistry,
	}
	if db, ok := drv.Connection().(*sql.DB); ok {
		o.db = db
	}
	return o, nil
}
