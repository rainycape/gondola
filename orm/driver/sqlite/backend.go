package sqlite

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/util/structs"
	"reflect"
	"time"
)

var (
	sqliteBackend    = &Backend{}
	transformedTypes = []reflect.Type{
		reflect.TypeOf((*time.Time)(nil)),
		reflect.TypeOf((*bool)(nil)),
	}
)

type Backend struct {
	sql.SqlBackend
}

func (b *Backend) Name() string {
	return "sqlite3"
}

func (b *Backend) Tag() string {
	return "sqlite"
}

func (b *Backend) FieldType(typ reflect.Type, t *structs.Tag) (string, error) {
	if c := codec.FromTag(t); c != nil {
		if c.Binary || t.PipeName() != "" {
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

func (b *Backend) FieldOptions(typ reflect.Type, t *structs.Tag) ([]string, error) {
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

func (b *Backend) ScanInt(val int64, goVal *reflect.Value, t *structs.Tag) error {
	switch goVal.Kind() {
	case reflect.Struct:
		goVal.Set(reflect.ValueOf(time.Unix(val, 0).UTC()))
		return nil
	case reflect.Bool:
		goVal.SetBool(val != 0)
		return nil
	}
	return b.SqlBackend.ScanInt(val, goVal, t)
}

func (b *Backend) TransformOutValue(val reflect.Value) (interface{}, error) {
	val = driver.Direct(val)
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

func sqliteOpener(url *config.URL) (driver.Driver, error) {
	return sql.NewDriver(sqliteBackend, url)
}

func init() {
	driver.Register("sqlite", sqliteOpener)
	driver.Register("sqlite3", sqliteOpener)
}
