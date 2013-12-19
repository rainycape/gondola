package template

import (
	"fmt"
	"reflect"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func index(item interface{}, indices ...interface{}) (interface{}, error) {
	v := reflect.ValueOf(item)
	if !v.IsValid() {
		return nil, nil
	}
	if k := v.Kind(); (k == reflect.Array || k == reflect.Slice || k == reflect.Map) && v.Len() == 0 {
		return nil, nil
	}
	for _, i := range indices {
		index := reflect.ValueOf(i)
		var isNil bool
		if v, isNil = indirect(v); isNil {
			return nil, fmt.Errorf("index of nil pointer")
		}
		switch v.Kind() {
		case reflect.Array, reflect.Slice, reflect.String:
			var x int64
			switch index.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				x = index.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				x = int64(index.Uint())
			default:
				return nil, fmt.Errorf("cannot index slice/array with type %s", index.Type())
			}
			if x < 0 || x >= int64(v.Len()) {
				return nil, fmt.Errorf("index out of range: %d", x)
			}
			v = v.Index(int(x))
		case reflect.Map:
			if !index.IsValid() {
				index = reflect.Zero(v.Type().Key())
			}
			if !index.Type().AssignableTo(v.Type().Key()) {
				return nil, fmt.Errorf("%s is not index type for %s", index.Type(), v.Type())
			}
			if x := v.MapIndex(index); x.IsValid() {
				v = x
			} else {
				v = reflect.Zero(v.Type().Elem())
			}
		default:
			return nil, fmt.Errorf("can't index item of type %s", v.Type())
		}
	}
	return v.Interface(), nil
}

func length(item interface{}) (int, error) {
	v, isNil := indirect(reflect.ValueOf(item))
	if isNil {
		return 0, fmt.Errorf("len of nil pointer")
	}
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len(), nil
	}
	return 0, fmt.Errorf("len of type %s", v.Type())
}

func call(fn interface{}, args ...interface{}) (interface{}, error) {
	v := reflect.ValueOf(fn)
	typ := v.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function of type %s", typ)
	}
	if !goodFunc(typ) {
		return nil, fmt.Errorf("function called with %d args; should be 1 or 2", typ.NumOut())
	}
	numIn := typ.NumIn()
	var dddType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return nil, fmt.Errorf("wrong number of args: got %d want at least %d", len(args), numIn-1)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return nil, fmt.Errorf("wrong number of args: got %d want %d", len(args), numIn)
		}
	}
	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		value := reflect.ValueOf(arg)
		// Compute the expected type. Clumsy because of variadics.
		var argType reflect.Type
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		} else {
			argType = dddType
		}
		if !value.IsValid() && canBeNil(argType) {
			value = reflect.Zero(argType)
		}
		if !value.Type().AssignableTo(argType) {
			return nil, fmt.Errorf("arg %d has type %s; should be %s", i, value.Type(), argType)
		}
		argv[i] = value
	}
	result := v.Call(argv)
	if len(result) == 2 && !result[1].IsNil() {
		return result[0].Interface(), result[1].Interface().(error)
	}
	return result[0].Interface(), nil
}

func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
		if v.Kind() == reflect.Interface && v.NumMethod() > 0 {
			break
		}
	}
	return v, false
}

func goodFunc(typ reflect.Type) bool {
	// We allow functions with 1 result or 2 results where the second is an error.
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == errorType:
		return true
	}
	return false
}

func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}
