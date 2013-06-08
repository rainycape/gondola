package sql

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func tryEncodeJson(typ reflect.Type) error {
	switch typ.Kind() {
	case reflect.Slice:
		return tryEncodeJson(typ.Elem())
	case reflect.Map:
		if err := tryEncodeJson(typ.Key()); err != nil {
			return err
		}
		return tryEncodeJson(typ.Elem())
	case reflect.Struct:
		// Check for no unexported fields
		for ii := 0; ii < typ.NumField(); ii++ {
			field := typ.Field(ii)
			if field.PkgPath != "" {
				return fmt.Errorf("%v contains unexported field %q", typ, field.Name)
			}
			if err := tryEncodeJson(field.Type); err != nil {
				return err
			}
		}
		return nil
	default:
		val := reflect.New(typ)
		_, err := json.Marshal(val.Interface())
		return err
	}
	panic("unreachable")
}
