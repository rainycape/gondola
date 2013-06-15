package driver

import (
	"fmt"
	"gondola/orm/tag"
	"reflect"
)

type Fields struct {
	// Lists the names of the fields in the database, in order
	Names []string
	// List the names of the qualified struct fields (e.g. Foo.Bar) in order
	QNames []string
	// Lists the indexes of the members (for FieldByIndex())
	Indexes [][]int
	// Fields which should be omitted when they are zero
	OmitZero []bool
	// Fields which should become null when they are zero
	NullZero []bool
	// Field types, in order
	Types []reflect.Type
	// Field tags, in order
	Tags []*tag.Tag
	// Maps db names to indexes
	NameMap map[string]int
	// Maps struct names to indexes
	QNameMap map[string]int
	// The index of the primary (-1 if there's no pk)
	PrimaryKey int
	// True if the primary key is an integer type with auto_increment
	IntegerAutoincrementPk bool
}

// Map takes a qualified struct name and returns its db name and type
func (f *Fields) Map(qname string) (dbName string, typ reflect.Type, err error) {
	if n, ok := f.QNameMap[qname]; ok {
		return f.Names[n], f.Types[n], nil
	}
	return "", nil, fmt.Errorf("can't map field %q to database name", qname)
}
