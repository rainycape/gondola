package sql

import (
	"bytes"
	"gnd.la/orm/driver"
	"gnd.la/orm/index"
	"gnd.la/util/structs"
	"reflect"
	"strings"
	"time"
)

// Backend is the interface implemented by drivers
// for database/sql orm backends
type Backend interface {
	// Name passsed to database/sql.Open
	Name() string
	// Tag returns the struct tag read by this backend
	Tag() string
	// Placeholder returns the placeholder for the n'th position
	Placeholder(int) string
	// Placeholders returns a placeholders string for the given number if parameters
	Placeholders(int) string
	// Insert performs an insert on the given database for the given model fields.
	// Most drivers should just return db.Exec(query, args...).
	Insert(DB, driver.Model, string, ...interface{}) (driver.Result, error)
	// Index creates and index if it doesn't exist using the provided model, index and name.
	Index(DB, driver.Model, *index.Index, string) error
	// Returns the db type of the given field (e.g. INTEGER)
	FieldType(reflect.Type, *structs.Tag) (string, error)
	// Returns the db options for the given field (.e.g PRIMARY KEY AUTOINCREMENT)
	FieldOptions(reflect.Type, *structs.Tag) ([]string, error)
	// Types that need to be transformed (e.g. sqlite transforms time.Time and bool to integer)
	Transforms() []reflect.Type
	// Scan an int64 from the db to Go
	ScanInt(val int64, goVal *reflect.Value, t *structs.Tag) error
	// Scan a float64 from the db to Go
	ScanFloat(val float64, goVal *reflect.Value, t *structs.Tag) error
	// Scan a bool from the db to Go
	ScanBool(val bool, goVal *reflect.Value, t *structs.Tag) error
	// Scan a []byte from the db to Go
	ScanByteSlice(val []byte, goVal *reflect.Value, t *structs.Tag) error
	// Scan a string from the db to Go
	ScanString(val string, goVal *reflect.Value, t *structs.Tag) error
	// Scan a *time.Time from the db to Go
	ScanTime(val *time.Time, goVal *reflect.Value, t *structs.Tag) error
	// Transform a value from Go to the database
	TransformOutValue(reflect.Value) (interface{}, error)
}

const placeholders = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"

type SqlBackend struct {
}

func (b *SqlBackend) Placeholder(n int) string {
	return "?"
}

func (b *SqlBackend) Placeholders(n int) string {
	p := placeholders
	if n > 32 {
		p = strings.Repeat("?,", n)
	}
	return p[:2*n-1]
}

func (b *SqlBackend) Insert(db DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	return db.Exec(query, args...)
}

func (b *SqlBackend) Index(db DB, m driver.Model, idx *index.Index, name string) error {
	var buf bytes.Buffer
	buf.WriteString("CREATE ")
	if idx.Unique {
		buf.WriteString("UNIQUE ")
	}
	buf.WriteString("INDEX IF NOT EXISTS ")
	buf.WriteString(name)
	buf.WriteString(" ON \"")
	buf.WriteString(m.Table())
	buf.WriteString("\" (")
	fields := m.Fields()
	for _, v := range idx.Fields {
		name, _, err := fields.Map(v)
		if err != nil {
			return err
		}
		buf.WriteString(name)
		if DescField(idx, v) {
			buf.WriteString(" DESC")
		}
		buf.WriteByte(',')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(")")
	_, err := db.Exec(buf.String())
	return err
}

func (b *SqlBackend) Transforms() []reflect.Type {
	return nil
}

// These Scan* methods always assume the type is right. Backends which might
// receive different types (e.g. a string like a []byte) should implement their
// own Scan* methods as required.

func (b *SqlBackend) ScanInt(val int64, goVal *reflect.Value, t *structs.Tag) error {
	goVal.SetInt(val)
	return nil
}

func (b *SqlBackend) ScanFloat(val float64, goVal *reflect.Value, t *structs.Tag) error {
	goVal.SetFloat(val)
	return nil
}

func (b *SqlBackend) ScanBool(val bool, goVal *reflect.Value, t *structs.Tag) error {
	goVal.SetBool(val)
	return nil
}

func (b *SqlBackend) ScanByteSlice(val []byte, goVal *reflect.Value, t *structs.Tag) error {
	if goVal.Kind() == reflect.String {
		goVal.SetString(string(val))
		return nil
	}
	if len(val) > 0 && !t.Has("raw") {
		b := make([]byte, len(val))
		copy(b, val)
		val = b
	}
	goVal.Set(reflect.ValueOf(val))
	return nil
}

func (b *SqlBackend) ScanString(val string, goVal *reflect.Value, t *structs.Tag) error {
	goVal.SetString(val)
	return nil
}

func (b *SqlBackend) ScanTime(val *time.Time, goVal *reflect.Value, t *structs.Tag) error {
	goVal.Set(reflect.ValueOf(*val))
	return nil
}

func (b *SqlBackend) TransformOutValue(val reflect.Value) (interface{}, error) {
	return val.Interface(), nil
}
