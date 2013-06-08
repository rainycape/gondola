package orm

import (
	"database/sql"
	"fmt"
	"gondola/defaults"
	"gondola/log"
	"gondola/orm/driver"
	ormsql "gondola/orm/drivers/sql"
	"gondola/orm/query"
	"reflect"
	"strings"
)

type Orm struct {
	driver  driver.Driver
	upserts bool
	logger  *log.Logger
	tags    string
	// this field is non-nil iff the ORM driver uses database/sql
	db *sql.DB
}

// Query returns a Query object, on which you can call
// Limit, Offset or Iter, to start iterating the results.
// The first argument is the Model object returned when
// registering the type.
func (o *Orm) Query(m *Model, q query.Q) *Query {
	return &Query{
		orm:    o,
		model:  m,
		q:      q,
		limit:  -1,
		offset: -1,
	}
}

// One fetchs the first result. The out parameter must be a pointer
// to an object previously registered as a model. If there are no
// results, ErrNotFound is returned.
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

// Insert saves an object into its collection. Its
// type must be previously registered as a model. If the model
// has an integer primary key with auto_increment, it will be
// be populated with the database assigned id.
func (o *Orm) Insert(obj interface{}) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	var pkName string
	var pkVal reflect.Value
	if m.fields.IntegerAutoincrementPk {
		pkName, pkVal = o.primaryKey(m.fields, obj)
		if pkVal.Int() == 0 && !pkVal.CanSet() {
			t := reflect.TypeOf(obj)
			return nil, fmt.Errorf("can't set primary key field %q. Please, insert a %v rather than a %v", pkName, reflect.PtrTo(t), t)
		}
	}
	res, err := o.driver.Insert(m, obj)
	if err == nil && pkVal.IsValid() && pkVal.Int() == 0 {
		id, err := res.LastInsertId()
		if err == nil && id != 0 {
			if o.logger != nil {
				o.logger.Debugf("Setting primary key %q to %d on model %v", pkName, id, m.typ)
			}
			pkVal.SetInt(id)
		} else if err != nil && o.logger != nil {
			o.logger.Errorf("could not obtain last insert id: %s", err)
		}
	}
	return res, err
}

// MustInsert works like insert, but panics if there's
// an error.
func (o *Orm) MustInsert(obj interface{}) Result {
	res, err := o.Insert(obj)
	if err != nil {
		panic(err)
	}
	return res
}

func (o *Orm) Update(obj interface{}, q query.Q) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	return o.driver.Update(m, obj, q)
}

// MustUpdate works like update, but panics if there's
// an error.
func (o *Orm) MustUpdate(obj interface{}, q query.Q) Result {
	res, err := o.Update(obj, q)
	if err != nil {
		panic(err)
	}
	return res
}

// Upsert tries to perform an update with the given query
// and object. If there are not affected rows, it performs
// an insert. Some drivers (like mongodb) are able to perform
// this operation in just one query, but most require two
// trips to the database.
func (o *Orm) Upsert(obj interface{}, q query.Q) (Result, error) {
	if o.upserts {
		m, err := o.model(obj)
		if err != nil {
			return nil, err
		}
		return o.driver.Upsert(m, obj, q)
	}
	res, err := o.Update(obj, q)
	if err != nil {
		return nil, err
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if aff == 0 {
		res, err = o.Insert(obj)
	}
	return res, err
}

// MustUpsert works like Upsert, but panics if there's an error.
func (o *Orm) MustUpsert(obj interface{}, q query.Q) Result {
	res, err := o.Upsert(obj, q)
	if err != nil {
		panic(err)
	}
	return res
}

// Save takes an object, with its type registered as
// a model with a primary key and either inserts it
// (if the primary key is zero or it has no primary key)
// or updates it using the primary key as the query
// (if it's non zero). If the update results in no
// affected rows, an insert will be performed.
func (o *Orm) Save(obj interface{}) (Result, error) {
	m, err := o.model(obj)
	if err != nil {
		return nil, err
	}
	if m.fields.PrimaryKey < 0 {
		return o.Insert(obj)
	}
	pkName, pkVal := o.primaryKey(m.fields, obj)
	if driver.IsZero(pkVal) {
		return o.Insert(obj)
	}
	res, err := o.Update(obj, Eq(pkName, pkVal.Interface()))
	if err != nil {
		return nil, err
	}
	up, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if up == 0 {
		return o.Insert(obj)
	}
	return res, nil
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

// Delete removes all objects from the given model matching
// the query.
func (o *Orm) Delete(m *Model, q query.Q) (Result, error) {
	return o.driver.Delete(m, q)
}

// DeleteObject removes the given object, which must be a type
// previously registered as a model and must have a primary key
func (o *Orm) DeleteObject(obj interface{}) error {
	return nil
}

// Close closes the database connection.
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
// returns nil.
func (o *Orm) SqlDB() *sql.DB {
	return o.db
}

// SqlQuery performs the given query on the database/sql backend and
// returns and iter with the results. If the underlying connection is
// not using database/sql, the returned Iter will have no results and
// will report the error ErrNoSql.
func (o *Orm) SqlQuery(m *Model, query string, args ...interface{}) Iter {
	if o.db == nil {
		return ormsql.NewIter(nil, nil, nil, ErrNoSql)
	}
	rows, err := o.db.Query(query, args...)
	return ormsql.NewIter(m, o.driver, rows, err)
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

func (o *Orm) model(obj interface{}) (*Model, error) {
	t := reflect.TypeOf(obj)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	model := _typeRegistry[o.tags][t]
	if model == nil {
		return nil, fmt.Errorf("no model registered for type %v with tags %q", t, o.tags)
	}
	return model, nil
}

func (o *Orm) primaryKey(f *driver.Fields, obj interface{}) (string, reflect.Value) {
	pk := f.PrimaryKey
	if pk < 0 {
		return "", reflect.Value{}
	}
	val := reflect.ValueOf(obj)
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return f.QNames[pk], val.FieldByIndex(f.Indexes[pk])
}

type sqldriver interface {
	DB() *sql.DB
}

func Open(name string, params string) (*Orm, error) {
	opener := driver.Get(name)
	if opener == nil {
		return nil, fmt.Errorf("no ORM driver named %q", name)
	}
	drv, err := opener(params)
	if err != nil {
		return nil, fmt.Errorf("Error opening %q driver: %s", name, err)
	}
	var db *sql.DB
	if dbDrv, ok := drv.(sqldriver); ok {
		db = dbDrv.DB()
	}
	return &Orm{
		driver:  drv,
		upserts: drv.Upserts(),
		tags:    strings.Join(drv.Tags(), "-"),
		db:      db,
	}, nil
}

func OpenDefault() (*Orm, error) {
	drv, source := defaults.DatabaseParameters()
	return Open(drv, source)
}
