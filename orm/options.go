package orm

import (
	"gondola/orm/driver"
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
	Name string
	// Any indexes that can't be declared using field tags
	// (most of the time because they index multiple fields
	// or require special flags). See the function Indexes()
	// for a convenience method for initializing this field.
	Indexes   []driver.Index
	Relations []*Relation
}
