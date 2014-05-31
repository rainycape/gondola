package sqlite

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"gnd.la/config"
	"gnd.la/encoding/codec"
	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/orm/index"
	"gnd.la/util/generic"
	"gnd.la/util/stringutil"
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

func (b *Backend) Inspect(db *sql.DB, m driver.Model) (*sql.Table, error) {
	name := db.QuoteString(m.Table())
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fieldsByName := make(map[string]*sql.Field)
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
		f.Type = strings.ToUpper(f.Type)
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
		fieldsByName[f.Name] = &f
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		field := fieldsByName[from]
		field.Constraints = append(field.Constraints, &sql.Constraint{Type: sql.ConstraintForeignKey, References: sql.MakeReference(table, to)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(fields) > 0 {
		return &sql.Table{Fields: fields}, nil
	}
	return nil, nil
}

func (b *Backend) HasIndex(db *sql.DB, m driver.Model, idx *index.Index, name string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA index_info(%s)", name))
	if err != nil {
		return false, err
	}
	has := rows.Next()
	rows.Close()
	return has, nil
}

func (b *Backend) DefineField(db *sql.DB, m driver.Model, table *sql.Table, field *sql.Field) (string, []string, error) {
	if field.HasOption(sql.OptionAutoIncrement) {
		if field.Constraint(sql.ConstraintPrimaryKey) == nil {
			return "", nil, fmt.Errorf("%s can only auto increment the primary key", b.Name())
		}
	}
	def, constraints, err := b.SqlBackend.DefineField(db, m, table, field)
	if err == nil {
		def = strings.Replace(strings.Replace(def, "DEFAULT false", "DEFAULT 0", -1), "DEFAULT true", "DEFAULT 1", -1)
	}
	return def, constraints, err
}

func (b *Backend) AddFields(db *sql.DB, m driver.Model, prevTable *sql.Table, newTable *sql.Table, fields []*sql.Field) error {
	rewrite := false
	for _, v := range fields {
		if !b.canAddField(v) {
			rewrite = true
			break
		}
	}
	if rewrite {
		name := db.QuoteIdentifier(m.Table())
		tmpName := fmt.Sprintf("%s_%s", m.Table(), stringutil.Random(8))
		quotedTmpName := db.QuoteIdentifier(tmpName)
		createSql, err := newTable.SQL(db, b, m, tmpName)
		if err != nil {
			return err
		}
		if _, err := db.Exec(createSql); err != nil {
			return err
		}
		fieldNames := generic.Map(prevTable.Fields, func(f *sql.Field) string { return f.Name }).([]string)
		// The previous table might have fields that we're not part
		// of the new table.
		fieldSet := make(map[string]bool)
		for _, v := range newTable.Fields {
			fieldSet[v.Name] = true
		}
		fieldNames = generic.Filter(fieldNames, func(n string) bool { return fieldSet[n] }).([]string)
		sqlFields := strings.Join(generic.Map(fieldNames, db.QuoteIdentifier).([]string), ", ")
		copySql := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s", quotedTmpName, sqlFields, sqlFields, name)
		if _, err := db.Exec(copySql); err != nil {
			return err
		}
		if _, err := db.Exec(fmt.Sprintf("DROP TABLE %s", name)); err != nil {
			return err
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quotedTmpName, name)); err != nil {
			return err
		}
		return nil
	}
	return b.SqlBackend.AddFields(db, m, prevTable, newTable, fields)
}

func (b *Backend) FieldType(typ reflect.Type, t *structs.Tag) (string, error) {
	if c := codec.FromTag(t); c != nil {
		if c.Binary || t.PipeName() != "" {
			return "BLOB", nil
		}
		return "TEXT", nil
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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

func (b *Backend) canAddField(f *sql.Field) bool {
	// These are a supeset of the actual resctrictions, for
	// simplicity. See https://www.sqlite.org/lang_altertable.html
	// for more details.
	return f.Constraint(sql.ConstraintPrimaryKey) != nil && f.Constraint(sql.ConstraintUnique) == nil && f.Default == ""
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
