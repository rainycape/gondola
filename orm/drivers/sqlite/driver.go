package sqlite

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gondola/orm/driver"
	"gondola/orm/drivers/sql"
	"reflect"
)

var sqliteBackend = &Backend{}

type Backend struct {
}

func (b *Backend) Name() string {
	return "sqlite3"
}

func (b *Backend) FieldType(typ reflect.Type, tag driver.Tag) (string, error) {
	switch typ.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER", nil
	case reflect.Float32, reflect.Float64:
		return "REAL", nil
	case reflect.String:
		return "TEXT", nil
	case reflect.Struct:
		if typ.Name() == "Time" && typ.PkgPath() == "time" {
			return "INT", nil
		}
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
	if tag.Has("auto_increment") {
		opts = append(opts, "AUTOINCREMENT")
	}
	if def := tag.Value("default"); def != "" {
		if typ.Kind() == reflect.String {
			def = "\"" + def + "\""
		}
		opts = append(opts, fmt.Sprintf("DEFAULT %s", def))
	}
	return opts, nil
}

func sqliteOpener(params string) (driver.Driver, error) {
	return sql.NewDriver(sqliteBackend, params)
}

func init() {
	driver.Register("sqlite", sqliteOpener)
	driver.Register("sqlite3", sqliteOpener)
}
