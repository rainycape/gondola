package generic

import (
	"fmt"
	"reflect"
)

type mapFunc func(handle) handle

func mapper(key string, typ reflect.Type) (mapFunc, reflect.Type, error) {
	fn, t, err := methodValue(key, typ)
	if err != nil {
		return nil, nil, err
	}
	if fn == nil {
		fn, t = fieldValue(key, typ)
	}
	if fn == nil {
		err = fmt.Errorf("%T does not have a field nor a method named %q", typ, key)
	}
	return fn, t, err
}

func methodValue(key string, typ reflect.Type) (mapFunc, reflect.Type, error) {
	if m, ok := typ.MethodByName(key); ok {
		if m.Type.NumIn() > 1 {
			return nil, nil, fmt.Errorf("method %s on type %s has %d arguments, must have none", key, typ, m.Type.NumIn()-1)
		}
		if m.Type.NumOut() != 1 {
			return nil, nil, fmt.Errorf("method %s on type %s returns %d values, must return one", key, typ, m.Type.NumOut())
		}
		return methodValueFunc(m), m.Type.Out(0), nil
	}
	return nil, nil, nil
}

func fieldValue(key string, typ reflect.Type) (mapFunc, reflect.Type) {
	depth := 0
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		depth++
	}
	if typ.Kind() == reflect.Struct {
		if field, ok := typ.FieldByName(key); ok {
			fn := fieldValueFunc(field, depth)
			if fn != nil {
				fdepth := 0
				ft := field.Type
				for ft.Kind() == reflect.Ptr {
					ft = ft.Elem()
					fdepth++
				}
				switch fdepth {
				case 0:
				case 1:
					f := fn
					fn = func(h handle) handle {
						return getElem(f(h))
					}
				default:
					f := fn
					fn = func(h handle) handle {
						v := f(h)
						for ii := 0; ii < fdepth; ii++ {
							v = getElem(v)
						}
						return h
					}
					fn = f
				}
				return fn, ft
			}
		}
	}
	return nil, nil
}

type lessFunc func(handle, handle) bool
type indexFunc func(handle, int) handle
type indexSetFunc func(handle, int, handle)
type swapFunc func(handle, int, int)

func lessComparator(t reflect.Type) (lessFunc, error) {
	if fn := getComparator(t); fn != nil {
		return fn, nil
	}
	return nil, fmt.Errorf("can't compare type %s", t)
}

func sliceMapper(sl interface{}, key string) (mapFunc, reflect.Value, reflect.Type, reflect.Type, error) {
	val := reflect.ValueOf(sl)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, reflect.Value{}, nil, nil, fmt.Errorf("invalid type %v, must be slice or array", val.Type())
	}
	length := val.Len()
	if length == 0 {
		return nil, reflect.Value{}, nil, nil, nil
	}
	elem := val.Type().Elem()
	fn, typ, err := mapper(key, elem)
	return fn, val, elem, typ, err
}
