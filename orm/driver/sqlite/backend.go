package sqlite

import (
	"fmt"
	"reflect"
	"time"

	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/orm/index"
	"gnd.la/util/structs"

	_ "github.com/mattn/go-sqlite3"
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

func (b *Backend) Func(fname string, retType reflect.Type) (string, error) {
	if fname == "now" && retType.PkgPath() == "time" && retType.Name() == "Time" {
		return "(strftime('%s', 'now'))", nil
	}
	return b.SqlBackend.Func(fname, retType)
}

func (b *Backend) Inspect(db sql.DB, m driver.Model) (*sql.Table, error) {
	name := db.QuoteString(m.Table())
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fields []*sql.Field
	for rows.Next() {
		var cid int
		var f sql.Field
		var notnull int
		var def *string
		var pk int
		if err := rows.Scan(&cid, &f.Name, &f.Type, &notnull, &def, &pk); err != nil {
			return nil, err
		}
		if notnull != 0 {
			f.AddConstraint(sql.ConstraintNotNull)
		}
		if def != nil {
			f.Default = *def
		}
		if pk != 0 {
			f.AddConstraint(sql.ConstraintPrimaryKey)
		}
		fields = append(fields, &f)
	}
	if len(fields) > 0 {
		return &sql.Table{Fields: fields}, nil
	}
	return nil, nil
}

func (b *Backend) HasIndex(db sql.DB, m driver.Model, idx *index.Index, name string) (bool, error) {
	rows, err := db.Query("PRAGMA index_info(?)", name)
	if err != nil {
		return false, err
	}
	has := rows.Next()
	rows.Close()
	return has, nil
}

func (b *Backend) DefineField(db sql.DB, m driver.Model, table *sql.Table, field *sql.Field) (string, error) {
	if field.HasOption(sql.OptionAutoIncrement) {
		if field.Constraint(sql.ConstraintPrimaryKey) == nil {
			return "", fmt.Errorf("%s can only auto increment the primary key", b.Name())
		}
	}
	return b.SqlBackend.DefineField(db, m, table, field)
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
	drv, err := sql.NewDriver(sqliteBackend, url)
	if err == nil {
		if _, err := drv.DB().Exec("PRAGMA foreign_keys = on"); err != nil {
			return nil, err
		}
	}
	return drv, err
}

func init() {
	driver.Register("sqlite", sqliteOpener)
	driver.Register("sqlite3", sqliteOpener)
}
