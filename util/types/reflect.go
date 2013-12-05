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
