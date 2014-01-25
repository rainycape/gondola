package config

import (
	"gnd.la/util/types"
	"reflect"
)

// BoolValue returns the boolean value of the field named
// by name, of def if the field does not exists or does
// not have the correct type.
func BoolValue(obj interface{}, name string, def bool) bool {
	v := getField(obj, name)
	if v.IsValid() && v.Kind() == reflect.Bool {
		return v.Bool()
	}
	return def
}

// IntValue returns the integer value of the field named
// by name, of def if the field does not exists or does
// not have the correct type.
func IntValue(obj interface{}, name string, def int) int {
	v := getField(obj, name)
	if v.IsValid() {
		if types.IsInt(v.Type()) {
			return int(v.Int())
		}
		if types.IsUint(v.Type()) {
			return int(v.Uint())
		}
	}
	return def
}

// StringValue returns the string value of the field named
// by name, of def if the field does not exists or does
// not have the correct type.
func StringValue(obj interface{}, name string, def string) string {
	v := getField(obj, name)
	if v.IsValid() && v.Kind() == reflect.String {
		return v.String()
	}
	return def
}

// PointerValue sets the out parameter, which must be a pointer to
// a pointer of the same type as the field, to point to the same
// object as the field named by name. The return value indicates
// if the assignment could be made.
func PointerValue(obj interface{}, name string, out interface{}) bool {
	o := reflect.ValueOf(out)
	if o.Kind() == reflect.Ptr && !o.IsNil() {
		el := o.Elem()
		v := getField(obj, name)
		if v.IsValid() && v.Type() == el.Type() {
			el.Set(v)
			return true
		}
	}
	return false
}

func getField(obj interface{}, name string) reflect.Value {
	v := reflect.ValueOf(obj)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		return v.FieldByName(name)
	}
	return reflect.Value{}
}
