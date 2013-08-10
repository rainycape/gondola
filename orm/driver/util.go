package driver

import (
	"reflect"
	"time"
)

var (
	timeType = reflect.TypeOf(time.Time{})
)

func IsZero(val reflect.Value) bool {
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
		if val.Type() == timeType {
			return val.Interface().(time.Time).IsZero()
		}
		return false
	}
	return true
}

// Direct descends into val.Elem() as long as
// val.Type().Kind() is reflect.Ptr and returns
// the result.
func Direct(val reflect.Value) reflect.Value {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}
