package index

type Index struct {
	// They fields indexed by this index, in order. For creating
	// indexes on nested structures, separate the names with a
	// dot. e.g. given the types:
	//    type Foo struct {
	//	    A int64
	//	    B int64
	//    }
	//    type Bar struct {
	//        Foo
	//    }
	// To create a index in Bar over A and B, you'd do:
	//    New("Foo.A", "Foo.B")
	Fields []string
	// Wheter the index should be unique.
	Unique  bool
	options map[int]interface{}
}

// Set sets a driver dependent option for the given index.
// The builtin drivers all use constants <= 10000, so if
// you're writing an ORM driver, use constants higher
// than that value. For conveniency, the options for the builtin
// drivers are defined as contants in this package.
// The same index is returned, to allow chaining calls.
func (i *Index) Set(opt int, values ...interface{}) *Index {
	if i.options == nil {
		i.options = make(map[int]interface{})
	}
	if len(values) == 1 {
		i.options[opt] = values[0]
	} else {
		i.options[opt] = values
	}
	return i
}

// Get returns the value for the given option.
func (i *Index) Get(opt int) interface{} {
	return i.options[opt]
}

// Indexes takes a variable number of indexes and returns
// them as a slice. It's intended as a convenience function
// for declaring model options.
func Indexes(indexes ...*Index) []*Index {
	return indexes
}

// New returns an index for the given fields. The names
// should be qualified Go names (e.g. Id or Foo.Id, not id or foo_id).
func New(fields ...string) *Index {
	return &Index{
		Fields: fields,
	}
}

// NewUnique returns an unique index for the given fields. The names
// should be qualified Go names (e.g. Id or Foo.Id, not id or foo_id).
func NewUnique(fields ...string) *Index {
	return &Index{
		Fields: fields,
		Unique: true,
	}
}
