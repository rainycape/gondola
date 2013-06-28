package postgres

import (
	"bytes"
	"fmt"
	_ "github.com/lib/pq"
	"gondola/orm/codec"
	"gondola/orm/driver"
	"gondola/orm/drivers/sql"
	"gondola/types"
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

func (b *Backend) Tag() string {
	return b.Name()
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

func (b *Backend) Insert(db sql.DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	fields := m.Fields()
	if fields.IntegerAutoincrementPk {
		query += " RETURNING " + fields.MNames[fields.PrimaryKey]
		var id int64
		err := db.QueryRow(query, args...).Scan(&id)
		if err != nil {
			return nil, err
		}
		return insertResult(id), nil
	}
	return db.Exec(query, args...)
}

func (b *Backend) Index(db sql.DB, m driver.Model, idx driver.Index, name string) error {
	// First, check if the index exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pg_class WHERE relname = $1", name).Scan(&count)
	if err == nil && count == 1 {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("CREATE ")
	if idx.Unique() {
		buf.WriteString("UNIQUE ")
	}
	buf.WriteString("INDEX ")
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
	_, err = db.Exec(buf.String())
	return err
}

func (b *Backend) FieldType(typ reflect.Type, t *types.Tag) (string, error) {
	if c := codec.FromTag(t); c != nil {
		// TODO: Use type JSON on Postgresql >= 9.2 for JSON encoded fields
		if c.Binary() {
			return "BYTEA", nil
		}
		return "TEXT", nil
	}
	var ft string
	switch typ.Kind() {
	case reflect.Bool:
		ft = "BOOL"
	case reflect.Int8, reflect.Uint8, reflect.Int16:
		ft = "INT2"
	case reflect.Uint16, reflect.Int32:
		ft = "INT4"
	case reflect.Int, reflect.Uint, reflect.Uint32, reflect.Int64, reflect.Uint64:
		ft = "INT8"
	case reflect.Float32:
		ft = "FLOAT4"
	case reflect.Float64:
		ft = "FLOAT8"
	case reflect.String:
		if t.Has("macaddr") {
			ft = "MACADDR"
		} else if t.Has("inet") {
			ft = "INET"
		} else {
			if ml := t.Value("max_length"); ml != "" {
				ft = fmt.Sprintf("VARCHAR (%s)", ml)
			} else {
				if fl := t.Value("length"); fl != "" {
					ft = fmt.Sprintf("CHAR (%s)", fl)
				} else {
					ft = "TEXT"
				}
			}
		}
	case reflect.Slice:
		etyp := typ.Elem()
		if etyp.Kind() == reflect.Uint8 {
			// []byte
			ft = "BYTEA"
			// TODO: database/sql does not support array types. Enable this code
			// if that changes in the future
			//		} else if typ.Elem().Kind() != reflect.Struct {
			//			et, err := b.FieldType(typ.Elem(), tag)
			//			if err != nil {
			//				return "", err
			//			}
			//			t = et + "[]"
		}
	case reflect.Struct:
		if typ.Name() == "Time" && typ.PkgPath() == "time" {
			ft = "TIMESTAMP WITHOUT TIME ZONE"
		}
	}
	if t.Has("auto_increment") {
		if strings.HasPrefix(ft, "INT") {
			ft = strings.Replace(ft, "INT", "SERIAL", -1)
		} else {
			return "", fmt.Errorf("postgres does not support auto incrementing %v", typ)
		}
	}
	if ft != "" {
		return ft, nil
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

func (b *Backend) ScanInt(val int64, goVal *reflect.Value, t *types.Tag) error {
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
	return sql.NewDriver(postgresBackend, params)
}

func init() {
	driver.Register("postgres", postgresOpener)
}
