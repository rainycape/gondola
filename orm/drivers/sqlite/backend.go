package sqlite

import (
	"bytes"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gondola/orm/codec"
	"gondola/orm/driver"
	"gondola/orm/drivers/sql"
	"gondola/types"
	"reflect"
	"strings"
	"time"
)

const placeholders = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"

var (
	sqliteBackend    = &Backend{}
	transformedTypes = []reflect.Type{
		reflect.TypeOf((*time.Time)(nil)),
		reflect.TypeOf((*bool)(nil)),
	}
)

type Backend struct {
}

func (b *Backend) Name() string {
	return "sqlite3"
}

func (b *Backend) Tag() string {
	return "sqlite"
}

func (b *Backend) Placeholder(n int) string {
	return "?"
}

func (b *Backend) Placeholders(n int) string {
	p := placeholders
	if n > 32 {
		p = strings.Repeat("?,", n)
	}
	return p[:2*n-1]
}

func (b *Backend) Insert(db sql.DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	return db.Exec(query, args...)
}

func (b *Backend) Index(db sql.DB, m driver.Model, idx driver.Index, name string) error {
	var buf bytes.Buffer
	buf.WriteString("CREATE ")
	if idx.Unique() {
		buf.WriteString("UNIQUE ")
	}
	buf.WriteString("INDEX IF NOT EXISTS ")
	buf.WriteString(name)
	buf.WriteString(" ON ")
	buf.WriteString(m.TableName())
	buf.WriteString(" (")
	fields := m.Fields()
	for _, v := range idx.Fields() {
		name, _, err := fields.Map(v)
		if err != nil {
			return err
		}
		buf.WriteString(name)
		if sql.DescField(idx, v) {
			buf.WriteString(" DESC")
		}
		buf.WriteByte(',')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(")")
	_, err := db.Exec(buf.String())
	return err
}

func (b *Backend) FieldType(typ reflect.Type, t *types.Tag) (string, error) {
	if c := codec.FromTag(t); c != nil {
		if c.Binary() {
			return "BLOB", nil
		}
		return "TEXT", nil
	}
	switch typ.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER", nil
	case reflect.Float32, reflect.Float64:
		return "REAL", nil
	case reflect.String:
		return "TEXT", nil
	case reflect.Slice:
		// []byte
		if typ.Elem().Kind() == reflect.Uint8 {
			return "BLOB", nil
		}
	case reflect.Struct:
		if typ.Name() == "Time" && typ.PkgPath() == "time" {
			return "INTEGER", nil
		}
	}
	return "", fmt.Errorf("can't map field type %v to a database type", typ)
}

func (b *Backend) FieldOptions(typ reflect.Type, t *types.Tag) ([]string, error) {
	var opts []string
	if t.Has("notnull") {
		opts = append(opts, "NOT NULL")
	}
	if t.Has("primary_key") {
		opts = append(opts, "PRIMARY KEY")
	} else if t.Has("unique") {
		opts = append(opts, "UNIQUE")
	}
	if t.Has("auto_increment") {
		if !t.Has("primary_key") {
			return nil, fmt.Errorf("%s can only auto increment the primary key", b.Name())
		}
		opts = append(opts, "AUTOINCREMENT")
	}
	if def := t.Value("default"); def != "" {
		if typ.Kind() == reflect.String {
			def = "'" + def + "'"
		}
		opts = append(opts, fmt.Sprintf("DEFAULT %s", def))
	}
	return opts, nil
}

func (b *Backend) Transforms() []reflect.Type {
	return transformedTypes
}

func (b *Backend) ScanInt(val int64, goVal *reflect.Value, t *types.Tag) error {
	switch goVal.Type().Kind() {
	case reflect.Struct:
		goVal.Set(reflect.ValueOf(time.Unix(val, 0).UTC()))
	case reflect.Bool:
		goVal.SetBool(val != 0)
	}
	return nil
}

func (b *Backend) ScanFloat(val float64, goVal *reflect.Value, t *types.Tag) error {
	return nil
}

func (b *Backend) ScanBool(val bool, goVal *reflect.Value, t *types.Tag) error {
	return nil
}

func (b *Backend) ScanByteSlice(val []byte, goVal *reflect.Value, t *types.Tag) error {
	return nil
}

func (b *Backend) ScanString(val string, goVal *reflect.Value, t *types.Tag) error {
	return nil
}

func (b *Backend) ScanTime(val *time.Time, goVal *reflect.Value, t *types.Tag) error {
	return nil
}

func (b *Backend) TransformOutValue(val reflect.Value) (interface{}, error) {
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	switch x := val.Interface().(type) {
	case time.Time:
		if x.IsZero() {
			return nil, nil
		}
		return x.Unix(), nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	}
	return nil, fmt.Errorf("can't transform type %v", val.Type())
}

func sqliteOpener(params string) (driver.Driver, error) {
	return sql.NewDriver(sqliteBackend, params)
}

func init() {
	driver.Register("sqlite", sqliteOpener)
	driver.Register("sqlite3", sqliteOpener)
}
