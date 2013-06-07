package sql

import (
	"database/sql"
	"gondola/orm/driver"
	"reflect"
)

// Backend is the interface implemented by drivers
// for database/sql orm backends
type Backend interface {
	// Name passsed to database/sql.Open
	Name() string
	// Placeholder returns the placeholder for the n'th position
	Placeholder(int) string
	// Placeholders returns a placeholders string for the given number if parameters
	Placeholders(int) string
	// Insert performs an insert on the given database for the given model fields.
	// Most drivers should just return db.Exec(query, args...).
	Insert(*sql.DB, driver.Model, string, ...interface{}) (driver.Result, error)
	// Returns the db type of the given field (e.g. INTEGER)
	FieldType(reflect.Type, driver.Tag) (string, error)
	// Returns the db options for the given field (.e.g PRIMARY KEY AUTOINCREMENT)
	FieldOptions(reflect.Type, driver.Tag) ([]string, error)
	// Types that need to be transformed (e.g. sqlite transforms time.Time to integers)
	Transforms() map[reflect.Type]reflect.Type
	// Transform a value from the database to Go
	TransformInValue(dbVal interface{}, goVal reflect.Value) error
	// Transform a value from Go to the database
	TransformOutValue(reflect.Value) (interface{}, error)
}
