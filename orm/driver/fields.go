package driver

import (
	"reflect"
)

type Fields struct {
	Names      []string
	Indexes    [][]int
	OmitNil    []bool
	NullZero   []bool
	Types      map[string]reflect.Type
	Tags       map[string]Tag
	PrimaryKey int
}
