package driver

type Index interface {
	// They fields indexed by thes index, in order. For creating
	// indexes on nested structures, separate the names with a
	// dot. e.g. given the types:
	//    type Foo struct {
	//	    A int64
	//	    B int64
	//    }
	//    type Bar struct {
	//        Foo
	//    }
	// To create a unique index in A and B, you'd do:
	//    UniqueIndex("Foo.A", "Foo.B")
	Fields() []string
	// Wheter the index should be unique.
	Unique() bool
	// Set sets a driver dependent option for the given index.
	// The builtin drivers all use constants <= 100000, so if
	// you're writing an ORM driver, use constants to be higher
	// than that value. For conveniency, the options for the builtin
	// drivers are defined as contants in the gondola/orm package.
	// The same index is returned, to allow chaining calls.
	Set(opt int, value interface{}) Index
	// Get returns the value for the given option.
	Get(opt int) interface{}
}
