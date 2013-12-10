package generic

import (
	"fmt"
	"reflect"
)

type mapFunc func(reflect.Value) reflect.Value

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
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if field, ok := typ.FieldByName(key); ok {
		return fieldValueFunc(field), field.Type
	}
	return nil, nil
}

type lessFunc func(reflect.Value, reflect.Value) bool

func lessComparator(t reflect.Type) (lessFunc, error) {
	switch t.Kind() {
	case reflect.Bool:
		return boolLess, nil
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return intLess, nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return uintLess, nil
	case reflect.Float32, reflect.Float64:
		return floatLess, nil
	case reflect.String:
		return stringLess, nil
	}
	return nil, fmt.Errorf("can't compare type %s", t)
}

func boolLess(a reflect.Value, b reflect.Value) bool {
	return !a.Bool() && b.Bool()
}

func intLess(a reflect.Value, b reflect.Value) bool {
	return a.Int() < b.Int()
}

func uintLess(a reflect.Value, b reflect.Value) bool {
	return a.Uint() < b.Uint()
}

func floatLess(a reflect.Value, b reflect.Value) bool {
	return a.Float() < b.Float()
}

func stringLess(a reflect.Value, b reflect.Value) bool {
	return a.String() < b.String()
}
