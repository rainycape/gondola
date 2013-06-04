package driver

import (
	"reflect"
)

type Model interface {
	Type() reflect.Type
	Collection() string
	Fields() *Fields
	FieldNames() []string
	FieldType(string) reflect.Type
	FieldTag(string) Tag
}
