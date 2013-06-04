package driver

import (
	"reflect"
)

type Fields struct {
	// Lists the names of the fields in the database, in order
	Names []string
	// Lists the indexes of the members (for FieldByIndex())
	Indexes [][]int
	// Fields which should be omitted when they are zero
	OmitZero []bool
	// Fields which should become null when they are zero
	NullZero []bool
	Types    map[string]reflect.Type
	Tags     map[string]Tag
	// Maps struct names to db names (e.g. Id => id, Foo.Bar => foo_bar)
	NameMap map[string]string
	// The index of the primary (-1 if there's no pk)
	PrimaryKey int
}
