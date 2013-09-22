package driver

import (
	"gnd.la/orm/index"
	"reflect"
)

type Model interface {
	Type() reflect.Type
	Table() string
	Fields() *Fields
	Indexes() []*index.Index
}
