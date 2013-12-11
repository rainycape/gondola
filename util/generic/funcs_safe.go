// +build appengine

package generic

import (
	"reflect"
)

type handle *reflect.Value

func indexValue(v handle, i int) handle {
	val := (*reflect.Value)(v).Index(i)
	return (handle)(&val)
}

func indexer(t reflect.Type) indexFunc {
	return indexValue
}

func swapValue(v handle, i, j int) {
	vi := (*reflect.Value)(v).Index(i)
	vj := (*reflect.Value)(v).Index(j)
	tmp := reflect.New(vi.Type()).Elem()
	tmp.Set(vi)
	vi.Set(vj)
	vj.Set(tmp)
}

func swapper(t reflect.Type) swapFunc {
	return swapValue
}

func fieldValueFunc(field reflect.StructField, depth int) mapFunc {
	idx := field.Index
	switch depth {
	case 0:
		return func(v handle) handle {
			val := (*reflect.Value)(v).FieldByIndex(idx)
			return (handle)(&val)
		}
	case 1:
		return func(v handle) handle {
			val := (*reflect.Value)(v).Elem().FieldByIndex(idx)
			return (handle)(&val)
		}
	default:
		return func(v handle) handle {
			rv := (*reflect.Value)(v)
			for ii := 0; ii < depth; ii++ {
				elem := rv.Elem()
				rv = &elem
			}
			val := rv.FieldByIndex(idx)
			return (handle)(&val)
		}
	}
}

func methodValueFunc(m reflect.Method) mapFunc {
	idx := m.Index
	return func(v handle) handle {
		res := (*reflect.Value)(v).Method(idx).Call(nil)[0]
		return (handle)(&res)
	}
}

func boolLess(a handle, b handle) bool {
	return !(*reflect.Value)(a).Bool() && (*reflect.Value)(b).Bool()
}

func intLess(a handle, b handle) bool {
	return (*reflect.Value)(a).Int() < (*reflect.Value)(b).Int()
}

func uintLess(a handle, b handle) bool {
	return (*reflect.Value)(a).Uint() < (*reflect.Value)(b).Uint()
}

func floatLess(a handle, b handle) bool {
	return (*reflect.Value)(a).Float() < (*reflect.Value)(b).Float()
}

func stringLess(a handle, b handle) bool {
	return (*reflect.Value)(a).String() < (*reflect.Value)(b).String()
}

func getHandle(val reflect.Value) handle {
	return handle(&val)
}

func getElem(h handle) handle {
	elem := (*reflect.Value)(h).Elem()
	return handle(&elem)
}

func getComparator(t reflect.Type) lessFunc {
	switch t.Kind() {
	case reflect.Bool:
		return boolLess
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return intLess
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return uintLess
	case reflect.Float32, reflect.Float64:
		return floatLess
	case reflect.String:
		return stringLess
	}
	return nil
}
