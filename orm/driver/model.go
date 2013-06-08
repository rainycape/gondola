package driver

import (
	"reflect"
)

type Model interface {
	Type() reflect.Type
	Collection() string
	Fields() *Fields
	Indexes() []Index
}
