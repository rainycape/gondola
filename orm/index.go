package orm

import (
	"gondola/orm/driver"
)

type index struct {
	fields  []string
	unique  bool
	options map[int]interface{}
}

func (i *index) Fields() []string {
	return i.fields
}

func (i *index) Unique() bool {
	return i.unique
}

func (i *index) Set(opt int, value interface{}) driver.Index {
	if i.options == nil {
		i.options = map[int]interface{}{}
	}
	i.options[opt] = value
	return i
}

func (i *index) Get(opt int) interface{} {
	return i.options[opt]
}

// Indexes takes a variable number of indexes and returns
// them as a slice. Is intended as a convenience function
// for declaring model options.
func Indexes(indexes ...driver.Index) []driver.Index {
	return indexes
}

// Index returns an index for the given fields. The names
// should be qualified Go names (e.g. Id or Foo.Id, not id or foo_id).
func Index(fields ...string) driver.Index {
	return &index{
		fields: fields,
	}
}

// UniqueIndex returns an unique index for the given fields. The names
// should be qualified Go names (e.g. Id or Foo.Id, not id or foo_id).
func UniqueIndex(fields ...string) driver.Index {
	return &index{
		fields: fields,
		unique: true,
	}
}
