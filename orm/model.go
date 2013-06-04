package orm

import (
	"gondola/orm/driver"
	"reflect"
)

func isNil(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Complex64, reflect.Complex128:
		return val.Complex() == 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		return val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return val.Uint() == 0
	case reflect.Struct:
		return false
	}
	return true
}

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

func (m *Model) Insert(data interface{}) (reflect.Value, []string, []interface{}, error) {
	// data is guaranteed to be of m.typ
	val := reflect.ValueOf(data)
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	fields := m.fields
	max := len(fields.Names)
	names := make([]string, 0, max)
	values := make([]interface{}, 0, max)
	for ii, v := range fields.Indexes {
		f := val.FieldByIndex(v)
		if fields.OmitNil[ii] && isNil(f) {
			continue
		}
		names = append(names, fields.Names[ii])
		values = append(values, f.Interface())
	}
	return val, names, values, nil
}

func (m *Model) Values(out interface{}) ([]interface{}, error) {
	return nil, nil
}
