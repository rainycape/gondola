package orm

import (
	"gondola/orm/driver"
	"reflect"
)

type model struct {
	options   *Options
	tableName string
	fields    *driver.Fields
	tags      string
}

func (m *model) Type() reflect.Type {
	return m.fields.Type
}

func (m *model) TableName() string {
	return m.tableName
}

func (m *model) Fields() *driver.Fields {
	return m.fields
}

func (m *model) Indexes() []driver.Index {
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
