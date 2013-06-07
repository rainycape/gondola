package postgres

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"gondola/orm/driver"
	ormsql "gondola/orm/drivers/sql"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const placeholders = "$1 ,$2 ,$3 ,$4 ,$5 ,$6 ,$7 ,$8 ,$9 ,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32"

var (
	postgresBackend  = &Backend{}
	transformedTypes = []reflect.Type{
		reflect.TypeOf((*time.Time)(nil)),
	}
)

type Backend struct {
}

func (b *Backend) Name() string {
	return "postgres"
}

func (b *Backend) Placeholder(n int) string {
	return "$" + strconv.Itoa(n)
}

func (b *Backend) Placeholders(n int) string {
	p := placeholders
	if n > 32 {
		p = b.makeplaceholders(n)
	}
	return p[:4*n-1]
}

func (b *Backend) Insert(db *sql.DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	fields := m.Fields()
	if fields.IntegerAutoincrementPk {
		query += " RETURNING " + fields.Names[fields.PrimaryKey]
		var id int64
		err := db.QueryRow(query, args...).Scan(&id)
		if err != nil {
			return nil, err
		}
		return insertResult(id), nil
	}
	return db.Exec(query, args...)
}

func (b *Backend) FieldType(typ reflect.Type, tag driver.Tag) (string, error) {
	var t string
	switch typ.Kind() {
	case reflect.Bool:
		t = "BOOL"
	case reflect.Int8, reflect.Uint8, reflect.Int16:
		t = "INT2"
	case reflect.Uint16, reflect.Int32:
		t = "INT4"
	case reflect.Int, reflect.Uint, reflect.Uint32, reflect.Int64, reflect.Uint64:
		t = "INT8"
	case reflect.Float32:
		t = "FLOAT4"
	case reflect.Float64:
		t = "FLOAT8"
	case reflect.String:
		if tag.Has("macaddr") {
			t = "MACADDR"
		} else if tag.Has("inet") {
			t = "INET"
		} else {
			if ml := tag.Value("max_length"); ml != "" {
				t = fmt.Sprintf("VARCHAR (%s)", ml)
			} else {
				if fl := tag.Value("length"); fl != "" {
					t = fmt.Sprintf("CHAR (%s)", fl)
				} else {
					t = "TEXT"
				}
			}
		}
	case reflect.Slice:
		fmt.Println(typ, typ.Elem())
		etyp := typ.Elem()
		if etyp.Kind() == reflect.Uint8 {
			// []byte
			t = "BYTEA"
		} else if tag.Has("json") {
			// TODO: Use type JSON on Postgresql >= 9.2
			t = "TEXT"
		} else {
			et, err := b.FieldType(typ.Elem(), tag)
			if err != nil {
				return "", err
			}
			t = et + "[]"
		}
	case reflect.Struct:
		if typ.Name() == "Time" && typ.PkgPath() == "time" {
			t = "TIMESTAMP WITHOUT TIME ZONE"
		}
	}
	if tag.Has("auto_increment") {
		if strings.HasPrefix(t, "INT") {
			t = strings.Replace(t, "INT", "SERIAL", -1)
		} else {
			return "", fmt.Errorf("postgres does not support auto incrementing %v", typ)
		}
	}
	if t != "" {
		return t, nil
	}
	return "", fmt.Errorf("can't map field type %v to a database type", typ)
}

func (b *Backend) FieldOptions(typ reflect.Type, tag driver.Tag) ([]string, error) {
	var opts []string
	if tag.Has("notnull") {
		opts = append(opts, "NOT NULL")
	}
	if tag.Has("primary_key") {
		opts = append(opts, "PRIMARY KEY")
	} else if tag.Has("unique") {
		opts = append(opts, "UNIQUE")
	}
	if def := tag.Value("default"); def != "" {
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

func (b *Backend) ScanInt(val int64, goVal *reflect.Value) error {
	return nil
}

func (b *Backend) ScanFloat(val float64, goVal *reflect.Value) error {
	return nil
}

func (b *Backend) ScanBool(val bool, goVal *reflect.Value) error {
	return nil
}

func (b *Backend) ScanByteSlice(val []byte, goVal *reflect.Value) error {
	return nil
}

func (b *Backend) ScanString(val string, goVal *reflect.Value) error {
	return nil
}

func (b *Backend) ScanTime(val *time.Time, goVal *reflect.Value) error {
	goVal.Set(reflect.ValueOf(val.UTC()))
	return nil
}

func (b *Backend) TransformOutValue(val reflect.Value) (interface{}, error) {
	return val.Interface().(time.Time).UTC(), nil
}

func (b *Backend) makeplaceholders(n int) string {
	var buf bytes.Buffer
	for ii := 1; ii <= n; ii++ {
		buf.WriteByte('$')
		buf.WriteString(strconv.Itoa(ii))
		buf.WriteByte(',')
	}
	return buf.String()
}

func postgresOpener(params string) (driver.Driver, error) {
	return ormsql.NewDriver(postgresBackend, params)
}

func init() {
	driver.Register("postgres", postgresOpener)
}
