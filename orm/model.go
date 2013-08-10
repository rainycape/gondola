package orm

import (
	"gondola/orm/driver"
	"gondola/orm/index"
	"reflect"
)

type model struct {
	options *Options
	table   string
	fields  *driver.Fields
	tags    string
}

func (m *model) Type() reflect.Type {
	return m.fields.Type
}

func (m *model) Table() string {
	return m.table
}

func (m *model) Fields() *driver.Fields {
	return m.fields
}

func (m *model) Indexes() []*index.Index {
	var indexes []*index.Index
	if m.options != nil {
		indexes = append(indexes, m.options.Indexes...)
	}
	// Add indexes declared in the fields
	for ii, v := range m.fields.Tags {
		if v.Has("index") {
			indexes = append(indexes, &index.Index{
				Fields: []string{m.fields.QNames[ii]},
				Unique: v.Has("unique"),
			})
		}
	}
	return indexes
}
