package sql

import (
	"gondola/orm/driver"
	"reflect"
)

// Backend is the interface implemented by drivers
// for database/sql orm backends
type Backend interface {
	// Name passsed to database/sql.Open
	Name() string
	// Returns the db type of the given field (e.g. INTEGER)
	FieldType(reflect.Type, driver.Tag) (string, error)
	// Returns the db options for the given field (.e.g PRIMARY KEY AUTOINCREMENT)
	FieldOptions(reflect.Type, driver.Tag) ([]string, error)
	// Types that need to be transformed (e.g. sqlite transforms time.Time to integers)
	Transforms() map[reflect.Type]reflect.Type
	// Transform a value from the database to Go
	TransformInValue(dbVal reflect.Value, goVal reflect.Value) error
	// Transform a value from Go to the database
	TransformOutValue(reflect.Value) (interface{}, error)
}
