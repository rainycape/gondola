package mysql

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/orm/driver/sqlite"
	"gnd.la/util/structs"
	"reflect"
	"time"
)

const placeholders = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"

var (
	mysqlBackend     = &Backend{}
	transformedTypes = []reflect.Type{
		reflect.TypeOf((*time.Time)(nil)),
		reflect.TypeOf((*int64)(nil)),
	}
)

type Backend struct {
	sqlite.Backend
}

func (b *Backend) Name() string {
	return "mysql"
}

func (b *Backend) Tag() string {
	return b.Name()
}

func (b *Backend) FieldType(typ reflect.Type, t *structs.Tag) (string, error) {
	if c := codec.FromTag(t); c != nil {
		if c.Binary || t.PipeName() != "" {
			return "BLOB", nil
		}
		return "TEXT", nil
	}
	var ft string
	switch typ.Kind() {
	case reflect.Bool:
		ft = "BOOL"
	case reflect.Int8:
		ft = "TINYINT"
	case reflect.Uint8:
		ft = "TINYINT UNSIGNED"
	case reflect.Int16:
		ft = "SMALLINT"
	case reflect.Uint16:
		ft = "SMALLINT UNSIGNED"
	case reflect.Int32:
		ft = "INT"
	case reflect.Uint32:
		ft = "INT UNSIGNED"
	case reflect.Int, reflect.Int64:
		ft = "BIGINT"
	case reflect.Uint, reflect.Uint64:
		ft = "BIGINT UNSIGNED"
	case reflect.Float32:
		ft = "FLOAT"
	case reflect.Float64:
		ft = "DOUBLE"
	case reflect.String:
		if ml, ok := t.IntValue("max_length"); ok {
			ft = fmt.Sprintf("VARCHAR (%d)", ml)
		} else if fl, ok := t.IntValue("length"); ok {
			ft = fmt.Sprintf("CHAR (%d)", fl)
		} else {
			ft = "TEXT"
		}
	case reflect.Slice:
		etyp := typ.Elem()
		if etyp.Kind() == reflect.Uint8 {
			// []byte
			ft = "BLOB"
		}
	case reflect.Struct:
		if typ.Name() == "Time" && typ.PkgPath() == "time" {
			ft = "DATETIME"
		}
	}
	if ft != "" {
		return ft, nil
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
		opts = append(opts, "AUTO_INCREMENT")
	}
	if def := t.Value("default"); def != "" {
		if typ.Kind() == reflect.String {
			def = "\"" + def + "\""
		}
		opts = append(opts, fmt.Sprintf("DEFAULT %s", def))
	}
	return opts, nil
}

func (b *Backend) Transforms() []reflect.Type {
	return transformedTypes
}

func (b *Backend) ScanInt(val int64, goVal *reflect.Value, t *structs.Tag) error {
	goVal.SetInt(val)
	return nil
}

func (b *Backend) ScanFloat(val float64, goVal *reflect.Value, t *structs.Tag) error {
	return nil
}

func (b *Backend) ScanBool(val bool, goVal *reflect.Value, t *structs.Tag) error {
	return nil
}

func (b *Backend) ScanByteSlice(val []byte, goVal *reflect.Value, t *structs.Tag) error {
	return nil
}

func (b *Backend) ScanString(val string, goVal *reflect.Value, t *structs.Tag) error {
	return nil
}

func (b *Backend) ScanTime(val *time.Time, goVal *reflect.Value, t *structs.Tag) error {
	goVal.Set(reflect.ValueOf(val.UTC()))
	return nil
}

func (b *Backend) TransformOutValue(val reflect.Value) (interface{}, error) {
	i := val.Interface()
	if t, ok := i.(time.Time); ok {
		return t.UTC(), nil
	}
	return i, nil
}

func mysqlOpener(url *config.URL) (driver.Driver, error) {
	url.Value += "?charset=UTF8&sql_mode=ANSI&parseTime=true&loc=UTC"
	return sql.NewDriver(mysqlBackend, url)
}

func init() {
	driver.Register("mysql", mysqlOpener)
}
