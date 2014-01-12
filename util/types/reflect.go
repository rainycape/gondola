package types

import (
	"errors"
	"reflect"
)

var (
	ErrCantSet = errors.New("can't set value (you might need to pass a pointer)")
)

func SettableValue(val interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(val)
	for v.Type().Kind() == reflect.Ptr {
		if !v.Elem().IsValid() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if !v.CanSet() {
		return reflect.Value{}, ErrCantSet
	}
	return v, nil
}

// IsNumeric returns true iff the type is one of
// the int, uint, float or complex types.
func IsNumeric(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Int || k == reflect.Uint ||
		k == reflect.Float64 || k == reflect.Float32 ||
		k == reflect.Int8 || k == reflect.Uint8 ||
		k == reflect.Int16 || k == reflect.Uint16 ||
		k == reflect.Int32 || k == reflect.Uint32 ||
		k == reflect.Int64 || k == reflect.Uint64 ||
		k == reflect.Complex128 || k == reflect.Complex64
}
