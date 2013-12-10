// +build !appengine

package generic

import (
	"reflect"
	"unsafe"
)

func fieldValueFunc(field reflect.StructField, depth int) mapFunc {
	typ := field.Type
	offset := field.Offset
	switch depth {
	case 0:
		return func(v reflect.Value) reflect.Value {
			p := v.Addr().Pointer() + offset
			return reflect.NewAt(typ, unsafe.Pointer(p)).Elem()
		}
	case 1:
		return func(v reflect.Value) reflect.Value {
			p := v.Pointer() + offset
			return reflect.NewAt(typ, unsafe.Pointer(p)).Elem()
		}
	default:
		depth--
		return func(v reflect.Value) reflect.Value {
			for ii := 0; ii < depth; ii++ {
				v = v.Elem()
			}
			p := v.Pointer() + offset
			return reflect.NewAt(typ, unsafe.Pointer(p)).Elem()
		}
	}
}

func methodValueFunc(m reflect.Method) mapFunc {
	idx := m.Index
	return func(v reflect.Value) reflect.Value {
		return v.Method(idx).Call(nil)[0]
	}
}
