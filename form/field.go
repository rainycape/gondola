package form

import (
	"gnd.la/types"
	"reflect"
)

type Field struct {
	Id          string
	Type        Type
	Label       string
	GoName      string
	Name        string
	Placeholder string
	Help        string

	addons []*AddOn
	value  reflect.Value
	s      *types.Struct
	sval   reflect.Value
	pos    int
	err    error
}

func (f *Field) Value() interface{} {
	return f.value.Interface()
}

func (f *Field) SettableValue() interface{} {
	return f.value.Addr().Interface()
}

func (f *Field) Tag() *types.Tag {
	return f.s.Tags[f.pos]
}

func (f *Field) HasAddOns() bool {
	return len(f.addons) > 0
}

func (f *Field) Err() error {
	return f.err
}
