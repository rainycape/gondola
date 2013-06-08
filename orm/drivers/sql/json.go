package sql

import (
	"encoding/json"
	"fmt"
	"gondola/orm/driver"
	"reflect"
)

func tryEncodeJson(typ reflect.Type, drv driver.Driver) error {
	switch typ.Kind() {
	case reflect.Slice:
		return tryEncodeJson(typ.Elem(), drv)
	case reflect.Map:
		if err := tryEncodeJson(typ.Key(), drv); err != nil {
			return err
		}
		return tryEncodeJson(typ.Elem(), drv)
	case reflect.Struct:
		// Check for unexported fields
		for ii := 0; ii < typ.NumField(); ii++ {
			field := typ.Field(ii)
			if field.PkgPath != "" {
				tag := driver.NewTag(field, drv)
				if tag.IsEmpty() || tag.Name() == "" {
					// Try json tag
					tag = driver.Tag(field.Tag.Get("json"))
				}
				if tag.Name() != "-" {
					return fmt.Errorf("%v contains unexported field %q. Tag it with the name \"-\" to explicitely ignore it.", typ, field.Name)
				}
			}
			if err := tryEncodeJson(field.Type, drv); err != nil {
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

func encodeJson(val reflect.Value) ([]byte, error) {
	return json.Marshal(val.Interface())
}

func decodeJson(data []byte, val *reflect.Value) error {
	return json.Unmarshal(data, val.Addr().Interface())
}
