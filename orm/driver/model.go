package driver

import (
	"gondola/orm/index"
	"reflect"
)

type Model interface {
	Type() reflect.Type
	Table() string
	Fields() *Fields
	Indexes() []*index.Index
}
