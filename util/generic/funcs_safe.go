package generic

import (
	"reflect"
)

func fieldValueFunc(field reflect.StructField, depth int) mapFunc {
	idx := field.Index
	switch depth {
	case 0:
		return func(v reflect.Value) reflect.Value {
			return v.FieldByIndex(idx)
		}
	case 1:
		return func(v reflect.Value) reflect.Value {
			return v.Elem().FieldByIndex(idx)
		}
	default:
		return func(v reflect.Value) reflect.Value {
			for ii := 0; ii < depth; ii++ {
				v = v.Elem()
			}
			return v.FieldByIndex(idx)
		}
	}
}

func methodValueFunc(m reflect.Method) mapFunc {
	idx := m.Index
	return func(v reflect.Value) reflect.Value {
		return v.Method(idx).Call(nil)[0]
	}
}
