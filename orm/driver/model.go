package driver

import (
	"reflect"
)

type Model interface {
	Type() reflect.Type
	TableName() string
	Fields() *Fields
	Indexes() []Index
}
