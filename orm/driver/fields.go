package driver

import (
	"fmt"
	"reflect"
)

type Fields struct {
	// Lists the names of the fields in the database, in order
	Names []string
	// List the names of the qualified struct fields (e.g. Foo.Bar) in order
	QualifiedNames []string
	// Lists the indexes of the members (for FieldByIndex())
	Indexes [][]int
	// Fields which should be omitted when they are zero
	OmitZero []bool
	// Fields which should become null when they are zero
	NullZero []bool
	// Name in db (e.g. id ) => Type
	Types    map[string]reflect.Type
	// Name in db (e.g. foo_bar) => Tag
	Tags     map[string]Tag
	// Maps struct names to db names (e.g. Id => id, Foo.Bar => foo_bar)
	NameMap map[string]string
	// The index of the primary (-1 if there's no pk)
	PrimaryKey int
	// True if the primary key is an integer type with auto_increment
	IntegerAutoincrementPk bool
}

// Map takes a qualified struct name and returns its db name and type
func (f *Fields) Map(qname string) (dbName string, typ reflect.Type, err error) {
    n, ok := f.NameMap[qname]
    if ok {
	return n, f.Types[n], nil
    }
    return "", nil, fmt.Errorf("can't map field %q to database name", n)
}
