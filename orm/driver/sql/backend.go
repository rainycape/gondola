package sql

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"gnd.la/orm/driver"
	"gnd.la/orm/index"
	"gnd.la/util/generic"
	"gnd.la/util/structs"
	"gnd.la/util/types"
)

// Backend is the interface implemented by drivers
// for database/sql orm backends
type Backend interface {
	// Check performs any required sanity checks on the connection.
	Check(*DB) error
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
	Inspect(*DB, driver.Model) (*Table, error)
	// HasIndex returns wheter an index exists using the provided model, index and name.
	HasIndex(*DB, driver.Model, *index.Index, string) (bool, error)
	// DefineField returns the complete field definition as a string, including name, type, options...
	// Field constraints are returned in the secon argument, each constraint should be an item in the
	// returned slice.
	DefineField(*DB, driver.Model, *Table, *Field) (string, []string, error)
	// AddFields adds the given field to the table for the given model. prevTable is the result
	// of Inspect() on the previous table, while newTable is generated from the model definition.
	AddFields(db *DB, m driver.Model, prevTable *Table, newTable *Table, fields []*Field) error
	// Alter field changes oldField to newField, potentially including the name.
	AlterField(db *DB, m driver.Model, table *Table, oldField *Field, newField *Field) error
	// Insert performs an insert on the given database for the given model fields.
	// Most drivers should just return db.Exec(query, args...).
	Insert(*DB, driver.Model, string, ...interface{}) (driver.Result, error)
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

func (b *SqlBackend) Check(_ *DB) error {
	return nil
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

func (b *SqlBackend) Inspect(db *DB, m driver.Model, schema string) (*Table, error) {
	var val int
	name := db.QuoteString(m.Table())
	s := db.QuoteString(schema)
	eq := fmt.Sprintf("SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE "+
		"TABLE_NAME = %s AND TABLE_SCHEMA = %s", name, s)
	err := db.QueryRow(eq).Scan(&val)
	if err != nil {
		if err == ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	// Select fields with their types
	iq := fmt.Sprintf("SELECT COLUMN_NAME, IS_NULLABLE, DATA_TYPE, "+
		"CHARACTER_MAXIMUM_LENGTH FROM INFORMATION_SCHEMA.COLUMNS "+
		"WHERE TABLE_NAME = %s AND TABLE_SCHEMA = %s", name, s)
	rows, err := db.Query(iq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fields []*Field
	fieldsByName := make(map[string]*Field)
	for rows.Next() {
		var f Field
		var nullable string
		var maxLength *int
		if err := rows.Scan(&f.Name, &nullable, &f.Type, &maxLength); err != nil {
			return nil, err
		}
		if maxLength != nil {
			f.Type = fmt.Sprintf("%s (%d)", f.Type, *maxLength)
		}
		f.Type = strings.ToUpper(f.Type)
		if nullable != "YES" {
			f.AddConstraint(ConstraintNotNull)
		}
		fields = append(fields, &f)
		fieldsByName[f.Name] = &f
	}
	// Field constraints
	cq := fmt.Sprintf("SELECT C.CONSTRAINT_NAME, CONSTRAINT_TYPE, COLUMN_NAME "+
		"FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS C JOIN "+
		"INFORMATION_SCHEMA.KEY_COLUMN_USAGE K ON C.CONSTRAINT_NAME = "+
		"K.CONSTRAINT_NAME WHERE C.TABLE_NAME = %s AND K.TABLE_NAME = %s "+
		"AND C.TABLE_SCHEMA = %s", name, name, s)
	rows, err = db.Query(cq)
	if err != nil {
		return nil, err
	}
	foreignKeys := make(map[string]string)
	defer rows.Close()
	for rows.Next() {
		var constraintName string
		var constraintType string
		var name string
		if err := rows.Scan(&constraintName, &constraintType, &name); err != nil {
			return nil, err
		}
		field := fieldsByName[name]
		if field == nil {
			return nil, fmt.Errorf("table %s has constraint on non-existing field %s", m.Table(), name)
		}
		switch strings.ToLower(constraintType) {
		case "primary key":
			field.AddConstraint(ConstraintPrimaryKey)
		case "foreign key":
			foreignKeys[constraintName] = name
		case "unique":
			field.AddConstraint(ConstraintUnique)
		default:
			return nil, fmt.Errorf("unknown constraint type %s on field %s in table %s", constraintType, name, m.Table())
		}
	}
	if len(foreignKeys) > 0 {
		// Resolve FKs
		fks := strings.Join(generic.Map(generic.Keys(foreignKeys).([]string), db.QuoteString).([]string), ", ")
		fq := fmt.Sprintf("SELECT CONSTRAINT_NAME, TABLE_NAME, COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE CONSTRAINT_NAME IN (%s)", fks)
		rows, err := db.Query(fq)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var constraintName string
			var tableName string
			var columnName string
			if err := rows.Scan(&constraintName, &tableName, &columnName); err != nil {
				return nil, err
			}
			fieldName := foreignKeys[constraintName]
			// Field was validated previously, won't be nil
			field := fieldsByName[fieldName]
			field.Constraints = append(field.Constraints, &Constraint{Type: ConstraintForeignKey, References: MakeReference(tableName, columnName)})
		}
	}
	return &Table{Fields: fields}, nil
}

func (b *SqlBackend) DefineField(db *DB, m driver.Model, table *Table, f *Field) (string, []string, error) {
	s := fmt.Sprintf("%s %s", db.QuoteIdentifier(f.Name), f.Type)
	if f.Constraint(ConstraintPrimaryKey) != nil && len(table.PrimaryKeys()) == 1 {
		s += " PRIMARY KEY"
	}
	if f.Constraint(ConstraintUnique) != nil {
		s += " UNIQUE"
	}
	if f.Constraint(ConstraintNotNull) != nil {
		s += " NOT NULL"
	}
	if f.HasOption(OptionAutoIncrement) {
		s += " AUTOINCREMENT"
	}
	if f.Default != "" {
		s += " DEFAULT " + f.Default
	}
	if ref := f.Constraint(ConstraintForeignKey); ref != nil {
		s += fmt.Sprintf(" REFERENCES %s(%s)",
			db.QuoteIdentifier(ref.References.Table()), db.QuoteIdentifier(ref.References.Field()))
	}
	return s, nil, nil
}

func (b *SqlBackend) AddFields(db *DB, m driver.Model, prevTable *Table, newTable *Table, fields []*Field) error {
	modelFields := m.Fields()
	tableName := db.QuoteIdentifier(m.Table())
	for _, v := range fields {
		idx := modelFields.MNameMap[v.Name]
		field := v
		hasDefault := modelFields.HasDefault(idx)
		if hasDefault && v.Constraint(ConstraintNotNull) != nil {
			// ORM level default
			// Must be added as nullable first, then the default value
			// must be set and finally the field has to be altered to be
			// nullable.
			field = field.Copy()
			var constraints []*Constraint
			for _, v := range field.Constraints {
				if v.Type != ConstraintNotNull {
					constraints = append(constraints, v)
				}
			}
			field.Constraints = constraints
		}
		sql, cons, err := field.SQL(db, m, newTable)
		if err != nil {
			return err
		}
		if _, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, sql)); err != nil {
			return err
		}
		if hasDefault {
			value := modelFields.DefaultValue(idx)
			fieldName := db.QuoteIdentifier(v.Name)
			if _, err := db.Exec(fmt.Sprintf("UPDATE %s SET %s = ?", tableName, fieldName), value); err != nil {
				return err
			}
			if v.Constraint(ConstraintNotNull) != nil {
				if err := db.Backend().AlterField(db, m, newTable, field, v); err != nil {
					return err
				}
			}
		}
		for _, c := range cons {
			if _, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s", tableName, c)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *SqlBackend) AlterField(db *DB, m driver.Model, table *Table, oldField *Field, newField *Field) error {
	return fmt.Errorf("SQL backend %s can't ALTER fields", db.Backend().Name())
}

func (b *SqlBackend) Insert(db *DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	return db.Exec(query, args...)
}

func (b *SqlBackend) Transforms() []reflect.Type {
	return nil
}

// These Scan* methods always assume the type is right. Backends which might
// receive different types (e.g. a string like a []byte) should implement their
// own Scan* methods as required.

func (b *SqlBackend) ScanInt(val int64, goVal *reflect.Value, t *structs.Tag) error {
	if types.Kind(goVal.Kind()) == types.Uint {
		goVal.SetUint(uint64(val))
		return nil
	}
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
