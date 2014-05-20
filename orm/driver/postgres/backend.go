package postgres

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/orm/index"
	"gnd.la/util/structs"

	_ "github.com/lib/pq"
)

const placeholders = "$1 ,$2 ,$3 ,$4 ,$5 ,$6 ,$7 ,$8 ,$9 ,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32"

var (
	postgresBackend  = &Backend{}
	transformedTypes = []reflect.Type{
		reflect.TypeOf((*time.Time)(nil)),
	}
)

type Backend struct {
	sql.SqlBackend
}

func (b *Backend) Name() string {
	return "postgres"
}

func (b *Backend) Tag() string {
	return b.Name()
}

func (b *Backend) Placeholder(n int) string {
	return "$" + strconv.Itoa(n+1)
}

func (b *Backend) Placeholders(n int) string {
	p := placeholders
	if n > 32 {
		p = b.makeplaceholders(n)
	}
	return p[:4*n-1]
}

func (b *Backend) Func(fname string, retType reflect.Type) (string, error) {
	if fname == "now" && retType.PkgPath() == "time" && retType.Name() == "Time" {
		return "(statement_timestamp() at time zone 'utc')", nil
	}
	return b.SqlBackend.Func(fname, retType)
}

func (b *Backend) Inspect(db *sql.DB, m driver.Model) (*sql.Table, error) {
	return b.SqlBackend.Inspect(db, m, "public")
}

func (b *Backend) DefineField(db *sql.DB, m driver.Model, table *sql.Table, field *sql.Field) (string, []string, error) {
	def, con, err := b.SqlBackend.DefineField(db, m, table, field)
	if err != nil {
		return "", nil, err
	}
	// AUTO INCREMENT in pgsql is provided via SERIAL type
	return strings.Replace(def, " AUTOINCREMENT", "", -1), con, nil
}

func (b *Backend) Insert(db *sql.DB, m driver.Model, query string, args ...interface{}) (driver.Result, error) {
	fields := m.Fields()
	if fields.AutoincrementPk {
		q := query + " RETURNING " + fields.MNames[fields.PrimaryKey]
		var id int64
		err := db.QueryRow(q, args...).Scan(&id)
		// We need to perform a "real" insert to find the real error, so
		// just let the code fall to the Exec at the end of the function
		// if there's an error.
		if err == nil {
			return insertResult(id), nil
		}
	}
	return db.Exec(query, args...)
}

func (b *Backend) HasIndex(db *sql.DB, m driver.Model, idx *index.Index, name string) (bool, error) {
	var exists int
	err := db.QueryRow("SELECT 1 FROM pg_class WHERE relname = $1 AND relkind = 'i'", name).Scan(&exists)
	if err == sql.ErrNoRows {
		err = nil
	}
	return exists != 0, err
}

func (b *Backend) FieldType(typ reflect.Type, t *structs.Tag) (string, error) {
	if c := codec.FromTag(t); c != nil {
		// TODO: Use type JSON on Postgresql >= 9.2 for JSON encoded fields
		if c.Binary || t.PipeName() != "" {
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
			if ml, ok := t.MaxLength(); ok {
				ft = fmt.Sprintf("VARCHAR (%d)", ml)
			} else if fl, ok := t.Length(); ok {
				ft = fmt.Sprintf("CHAR (%d)", fl)
			} else {
				ft = "TEXT"
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

func (b *Backend) Transforms() []reflect.Type {
	return transformedTypes
}

func (b *Backend) ScanTime(val *time.Time, goVal *reflect.Value, t *structs.Tag) error {
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

func postgresOpener(url *config.URL) (driver.Driver, error) {
	return sql.NewDriver(postgresBackend, url)
}

func init() {
	driver.Register("postgres", postgresOpener)
}
