package orm

import (
	"gnd.la/orm/index"
)

// Options are the structure used to specify
// further options when registing a model.
type Options struct {
	// The name to use when creating for the table
	// or collection. If no name is provided, it will
	// be derived from the package and type name.
	// Type Baz in package foo/bar will be named
	// foo_bar_baz, but types in the main package will
	// omit the package name (e.g. Foo becomes foo,
	// not main_foo).
	Table string
	// Name is the model name which is going to be registered
	// by default, it's the type name in its package. The
	// model name is used when specifying relations and when
	// disambiguating field names.
	Name string
	// Any indexes that can't be declared using field tags
	// (most of the time because they index multiple fields
	// or require special flags).
	Indexes []*index.Index
	// The primary key when it's not specified in struct tags
	// or when it's a composite key. The names of the fields
	// must be qualified names (i.e. the name of the field
	// in the Go struct). If the primary key is
	// defined in both the a field tag and using this field, an
	// error will be returned when registering the model.
	PrimaryKey []string
	// Default indicates if the model should override any previously
	// registered models and become the default model for its type.
	// (otherwise, the first registered model is the default for the type).
	Default bool
}
