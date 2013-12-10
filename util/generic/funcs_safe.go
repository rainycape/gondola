package generic

import (
	"reflect"
)

func fieldValueFunc(field reflect.StructField) mapFunc {
	idx := field.Index
	return func(v reflect.Value) reflect.Value {
		for v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		return v.FieldByIndex(idx)
	}
}

func methodValueFunc(m reflect.Method) mapFunc {
	idx := m.Index
	return func(v reflect.Value) reflect.Value {
		return v.Method(idx).Call(nil)[0]
	}
}
