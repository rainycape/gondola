package sql

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"gnd.la/orm/driver"
	"gnd.la/orm/index"
	"gnd.la/util/structs"
)

// Backend is the interface implemented by drivers
// for database/sql orm backends
type Backend interface {
	// Name passsed to database/sql.Open
	Name() string
	// Tag returns the struct tag read by this backend
	Tag() string
	// Capabilities returns the backend capabilities not provided
	// by the SQL driver itself.
	Capabilities() driver.Capability
	// Placeholder returns the placeholder for the n'th position
	Placeholder(int) string
	// Placeholders returns a placeholders string for the given number if parameters
	Placeholders(int) string
	// StringQuote returns the character used for quoting strings.
	StringQuote() byte
	// IdentifierQuote returns the character used for quoting identifiers.
	IdentifierQuote() byte
	// Func returns the function which corresponds to the given name and
	// return type at the database level.
	Func(string, reflect.Type) (string, error)
	// DefaultValues returns the string used to signal that a INSERT has no provided
	// values and the default ones should be used.
	DefaultValues() string
	// Inspect returns the table as it exists in the database for the current model. If
	// the table does not exist, the Backend is expected to return (nil, nil).
	Inspect(DB, driver.Model) (*Table, error)
	// HasIndex returns wheter an index exists using the provided model, index and name.
	HasIndex(DB, driver.Model, *index.Index, string) (bool, error)
	// DefineField returns the complete field definition as a string, including name, type, options... etc.
	DefineField(DB, driver.Model, *Table, *Field) (string, error)
	AddField(DB, driver.Model, *Table, *Field) error
	// Insert performs an insert on the given database for the given model fields.
	// Most drivers should just return db.Exec(query, args...).
	Insert(DB, driver.Model, string, ...interface{}) (driver.Result, error)
	// Returns the db type of the given field (e.g. INTEGER)
	FieldType(reflect.Type, *structs.Tag) (string, error)
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

func (b *SqlBackend) Capabilities() driver.Capability {
	return driver.CAP_DEFAULTS_TEXT
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

func (b *SqlBackend) StringQuote() byte {
	return '\''
}

func (b *SqlBackend) IdentifierQuote() byte {
	return '"'
}

func (b *SqlBackend) Func(fname string, retType reflect.Type) (string, error) {
	return "", ErrFuncNotSupported
}

func (b *SqlBackend) DefaultValues() string {
	return "DEFAULT VALUES"
}

func (b *SqlBackend) Inspect(db DB, m driver.Model, schema string) (*Table, error) {
	var val int
	name := db.QuoteString(m.Table())
	s := db.QuoteString(schema)
	eq := fmt.Sprintf("SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = %s AND TABLE_SCHEMA = %s", name, s)
	err := db.QueryRow(eq).Scan(&val)
	if err != nil {
		if err == ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	iq := fmt.Sprintf("SELECT COLUMN_NAME, IS_NULLABLE, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = %s AND TABLE_SCHEMA = %s", name, s)
	rows, err := db.Query(iq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fields []*Field
	for rows.Next() {
		var f Field
		var nullable string
		var maxLength *int
		if err := rows.Scan(&f.Name, &nullable, &f.Type, &maxLength); err != nil {
			return nil, err
		}
		if nullable != "YES" {
			f.AddConstraint(ConstraintNotNull)
		}
		fields = append(fields, &f)
	}
	return &Table{Fields: fields}, nil
}

func (b *SqlBackend) DefineField(db DB, m driver.Model, table *Table, f *Field) (string, error) {
	s := fmt.Sprintf("%s %s", db.QuoteIdentifier(f.Name), f.Type)
	if f.Constraint(ConstraintPrimaryKey) != nil {
		// Otherwise it's added as a constraint
		if len(table.PrimaryKeys()) == 1 {
			s += " PRIMARY KEY"
		}
	} else {
		if f.Constraint(ConstraintUnique) != nil {
			s += " UNIQUE"
		}
		if f.Constraint(ConstraintNotNull) != nil {
			s += " NOT NULL"
		}
	}
	if f.HasOption(OptionAutoIncrement) {
		s += " AUTOINCREMENT"
	}
	if f.Default != "" {
		s += " DEFAULT " + f.Default
	}
	return s, nil
}

func (b *SqlBackend) AddField(DB, driver.Model, *Table, *Field) error {
	return fmt.Errorf("can't add field!")
}

func (b *SqlBackend) Insert(db DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	return db.Exec(query, args...)
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
