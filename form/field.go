package form

import (
	"fmt"
	"gnd.la/i18n"
	"gnd.la/util/structs"
	"reflect"
)

type Field struct {
	Type        Type
	Name        string
	HTMLName    string
	Label       i18n.String
	Placeholder i18n.String
	Help        i18n.String

	id     string
	prefix string
	addons []*AddOn
	value  reflect.Value
	s      *structs.Struct
	sval   reflect.Value
	pos    int
	err    error
}

func (f *Field) String() string {
	if f.err != nil {
		return fmt.Sprintf("%s=%s - error %s", f.HTMLName, f.Value(), f.err)
	}
	return fmt.Sprintf("%s=%s", f.HTMLName, f.Value())
}

func (f *Field) Id() string {
	return f.prefix + f.id
}

func (f *Field) Value() interface{} {
	return f.value.Interface()
}

func (f *Field) SettableValue() interface{} {
	return f.value.Addr().Interface()
}

func (f *Field) Tag() *structs.Tag {
	return f.s.Tags[f.pos]
}

func (f *Field) HasAddOns() bool {
	return len(f.addons) > 0
}

func (f *Field) Err() error {
	return f.err
}
