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
	driver     driver.Driver
	upserts    bool
	logger     *log.Logger
	tags       string
	numQueries int
	// this field is non-nil iff the ORM driver uses database/sql
	db *sql.DB
}

// Table returns a Query object initialized with the given table.
// The Table object is returned when registering the model. See
// the explanation on the Query's Table method about why strings
// are not accepted.
func (o *Orm) Table(t *Table) *Query {
	return &Query{
		orm:    o,
		model:  t.model,
		limit:  -1,
		offset: -1,
	}
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
func (o *Orm) One(q query.Q, out interface{}) error {
	return o.Query(q).One(out)
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

// MustInsert works like insert, but panics if there's
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
	return o.insert(t.model, obj)
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
	var pkName string
	var pkVal reflect.Value
	f := m.fields
	if f.IntegerAutoincrementPk {
		pkName, pkVal = o.primaryKey(f, obj)
		if pkVal.Int() == 0 && !pkVal.CanSet() {
			typ := reflect.TypeOf(obj)
			return nil, fmt.Errorf("can't set primary key field %q. Please, insert a %v rather than a %v", pkName, reflect.PtrTo(typ), typ)
		}
	}
	o.numQueries++
	res, err := o.driver.Insert(m, obj)
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
	o.numQueries++
	return o.driver.Update(m, q, obj)
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
	if o.upserts {
		o.numQueries++
		return o.driver.Upsert(m, q, obj)
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
// affected rows, an insert will be performed.
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
	return o.save(t.model, obj)
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
	if m.fields.PrimaryKey < 0 {
		return o.Insert(obj)
	}
	pkName, pkVal := o.primaryKey(m.fields, obj)
	if driver.IsZero(pkVal) {
		return o.insert(m, obj)
	}
	res, err := o.update(m, Eq(pkName, pkVal.Interface()), obj)
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
	return o.delete(t.model, q)
}

// Delete removes the given object, which must be of a type
// previously registered as a table and must have a primary key
func (o *Orm) Delete(obj interface{}) error {
	m, err := o.model(obj)
	if err != nil {
		return err
	}
	return o.deleteByPk(m, obj)
}

// DeleteFrom works like Delete, but deletes from the given table
// (as returned by Register)
func (o *Orm) DeleteFrom(t *Table, obj interface{}) error {
	return o.deleteByPk(t.model, obj)
}

func (o *Orm) deleteByPk(m *model, obj interface{}) error {
	pkName, pkVal := o.primaryKey(m.fields, obj)
	if !pkVal.IsValid() || pkName == "" {
		return fmt.Errorf("type %T does not have a primary key", obj)
	}
	q := Eq(pkName, pkVal.Interface())
	_, err := o.delete(m, q)
	return err
}

func (o *Orm) delete(m *model, q query.Q) (Result, error) {
	o.numQueries++
	return o.driver.Delete(m, q)
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

// NumQueries returns the number of queries since the ORM was
// initialized. Keep in mind that this number might not be
// completely in all cases accurate, since drivers are free
// to perform several queries per operation. However, the numbers
// reported by SQL drivers are currently accurate.
func (o *Orm) NumQueries() int {
	return o.numQueries
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
func (o *Orm) SqlQuery(t *Table, query string, args ...interface{}) *Iter {
	if o.db == nil {
		return &Iter{err: ErrNoSql}
	}
	rows, err := o.db.Query(query, args...)
	return &Iter{
		Iter: ormsql.NewIter(t.model, o.driver, rows, err),
		q:    &Query{model: t.model},
	}
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

func (o *Orm) model(obj interface{}) (*model, error) {
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
