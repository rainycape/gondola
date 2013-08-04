package driver

import (
	"gondola/types"
)

type Fields struct {
	*types.Struct
	// Fields which should be omitted when they are empty
	OmitEmpty []bool
	// Fields which should become null when they are empty
	NullEmpty []bool
	// The index of the primary (-1 if there's no pk)
	PrimaryKey int
	// True if the primary key is an integer type with auto_increment
	IntegerAutoincrementPk bool
	// Model methods called by the ORM
	Methods Methods
}
