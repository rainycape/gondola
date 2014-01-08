package template

import (
	"fmt"
	"gnd.la/util/types"
	"reflect"
	"strings"
)

func lookup(v reflect.Value, key string) (reflect.Value, error) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return reflect.Value{}, fmt.Errorf("can't lookup maps with non-string keys (%s)", v.Type().Key())
		}
		val := v.MapIndex(reflect.ValueOf(key))
		if !val.IsValid() {
			var keys []string
			for _, mk := range v.MapKeys() {
				keys = append(keys, fmt.Sprintf("%q", mk.String()))
				return reflect.Value{}, fmt.Errorf("map does not contain key %q (keys are %s)", key, strings.Join(keys, ", "))
			}
		}
		return val, nil
	case reflect.Struct:
		val := v.FieldByName(key)
		if !val.IsValid() {
			return reflect.Value{}, fmt.Errorf("type %s does not a have a field name %q", v.Type(), key)
		}
		return val, nil
	}
	return reflect.Value{}, fmt.Errorf("can't lookup field on type %v", v.Type())
}

func eval(obj interface{}, varname string) (string, error) {
	k := varname
	v := reflect.ValueOf(obj)
	dot := strings.IndexByte(k, '.')
	if dot >= 0 {
		k = k[:dot]
	}
	if k == "Vars" {
		return eval(obj, varname[dot+1:])
	}
	res, err := lookup(v, k)
	if err != nil {
		return "", err
	}
	if dot >= 0 {
		return eval(res.Interface(), varname[dot+1:])
	}
	return types.ToString(res.Interface()), nil
}
