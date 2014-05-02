package mysql

import (
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
	"gnd.la/util/types"

	_ "github.com/go-sql-driver/mysql"
)

const placeholders = "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?"

var (
	mysqlBackend     = &Backend{}
	transformedTypes = []reflect.Type{
		reflect.TypeOf((*time.Time)(nil)),
	}
)

type Backend struct {
	sql.SqlBackend
}

func (b *Backend) Name() string {
	return "mysql"
}

func (b *Backend) Tag() string {
	return b.Name()
}

func (b *Backend) Capabilities() driver.Capability {
	return driver.CAP_NONE
}

func (b *Backend) DefaultValues() string {
	return "() VALUES()"
}

func (b *Backend) Inspect(db *sql.DB, m driver.Model) (*sql.Table, error) {
	var database string
	if err := db.QueryRow("SELECT DATABASE() FROM DUAL").Scan(&database); err != nil {
		return nil, err
	}
	return b.SqlBackend.Inspect(db, m, database)
}

func (b *Backend) DefineField(db *sql.DB, m driver.Model, table *sql.Table, field *sql.Field) (string, []string, error) {
	def, cons, err := b.SqlBackend.DefineField(db, m, table, field)
	if err != nil {
		return "", nil, err
	}
	if ref := field.Constraint(sql.ConstraintForeignKey); ref != nil {
		if pos := strings.Index(def, " REFERENCES"); pos >= 0 {
			def = def[:pos]
		}
		refTable := ref.References.Table()
		refField := ref.References.Field()
		fkName := db.QuoteIdentifier(fmt.Sprintf("%s_%s_%s_%s", m.Table(), field.Name, refTable, refField))
		cons = append(cons, fmt.Sprintf("FOREIGN KEY %s(%s) REFERENCES %s(%s)", fkName, db.QuoteIdentifier(field.Name),
			db.QuoteIdentifier(refTable), db.QuoteIdentifier(refField)))
	}
	return strings.Replace(def, "AUTOINCREMENT", "AUTO_INCREMENT", -1), cons, nil
}

func (b *Backend) AlterField(db *sql.DB, m driver.Model, table *sql.Table, oldField *sql.Field, newField *sql.Field) error {
	fsql, cons, err := newField.SQL(db, m, table)
	if err != nil {
		return err
	}
	tableName := db.QuoteIdentifier(m.Table())
	if _, err = db.Exec(fmt.Sprintf("ALTER TABLE %s CHANGE COLUMN %s %s", tableName, db.QuoteIdentifier(oldField.Name), fsql)); err != nil {
		return err
	}
	for _, c := range cons {
		if _, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s", tableName, c)); err != nil {
			return err
		}
	}
	return err
}

func (b *Backend) HasIndex(db *sql.DB, m driver.Model, idx *index.Index, name string) (bool, error) {
	rows, err := db.Query("SHOW INDEX FROM ? WHERE Key_name = ?", m.Table(), name)
	if err != nil {
		return false, err
	}
	has := rows.Next()
	rows.Close()
	return has, nil
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
		if ml, ok := t.MaxLength(); ok {
			ft = fmt.Sprintf("VARCHAR (%d)", ml)
		} else if fl, ok := t.Length(); ok {
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

func (b *Backend) Transforms() []reflect.Type {
	return transformedTypes
}

func (b *Backend) ScanByteSlice(val []byte, goVal *reflect.Value, t *structs.Tag) error {
	// mysql returns u?int types as []byte under
	// some circumstances (not sure exactly when, but other
	// times they're returned as an int64).
	switch types.Kind(goVal.Kind()) {
	case types.Int:
		v, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return err
		}
		goVal.SetInt(v)
		return nil
	case types.Uint:
		v, err := strconv.ParseUint(string(val), 10, 64)
		if err != nil {
			return err
		}
		goVal.SetUint(v)
		return nil
	}
	return b.SqlBackend.ScanByteSlice(val, goVal, t)
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
	url.Query["charset"] = "UTF8"
	url.Query["sql_mode"] = "ANSI"
	url.Query["parseTime"] = "true"
	url.Query["loc"] = "UTC"
	url.Query["clientFoundRows"] = "true"
	return sql.NewDriver(mysqlBackend, url)
}

func init() {
	driver.Register("mysql", mysqlOpener)
}
