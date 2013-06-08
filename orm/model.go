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

func (m *Model) Indexes() []driver.Index {
	var indexes []driver.Index
	if m.options != nil {
		indexes = append(indexes, m.options.Indexes...)
	}
	// Add indexes declared in the fields
	for ii, v := range m.fields.Tags {
		if v.Has("index") {
			indexes = append(indexes, &index{
				fields: []string{m.fields.QNames[ii]},
				unique: v.Has("unique"),
			})
		}
	}
	return indexes
}
