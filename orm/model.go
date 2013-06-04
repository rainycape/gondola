package orm

import (
	"gondola/orm/driver"
	"reflect"
)

type Model struct {
	typ        reflect.Type
	options    *Options
	collection string
	fields     *driver.Fields
}

func (m *Model) Type() reflect.Type {
	return m.typ
}

func (m *Model) Collection() string {
	return m.collection
}

func (m *Model) Fields() *driver.Fields {
	return m.fields
}

func (m *Model) FieldNames() []string {
	return m.fields.Names
}

func (m *Model) FieldType(name string) reflect.Type {
	return m.fields.Types[name]
}

func (m *Model) FieldTag(name string) driver.Tag {
	return m.fields.Tags[name]
}
