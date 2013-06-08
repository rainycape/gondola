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
	tags       string
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
