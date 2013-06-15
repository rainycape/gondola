package codec

import (
	"encoding/json"
	"fmt"
	"gondola/orm/tag"
	"reflect"
)

type jsonCodec struct {
}

func (c *jsonCodec) Name() string {
	return "json"
}

func (c *jsonCodec) Binary() bool {
	return false
}

func (c *jsonCodec) Try(typ reflect.Type, tags []string) error {
	switch typ.Kind() {
	case reflect.Slice:
		return c.Try(typ.Elem(), tags)
	case reflect.Map:
		if err := c.Try(typ.Key(), tags); err != nil {
			return err
		}
		return c.Try(typ.Elem(), tags)
	case reflect.Struct:
		// Check for unexported fields
		for ii := 0; ii < typ.NumField(); ii++ {
			field := typ.Field(ii)
			if field.PkgPath != "" {
				t := tag.New(field, tags)
				if t.IsEmpty() || t.Name() == "" {
					// Try json tag
					t = tag.NewNamed(field, "json")
				}
				if t.Name() != "-" {
					return fmt.Errorf("%v contains unexported field %q. Tag it with the name \"-\" to explicitely ignore it.", typ, field.Name)
				}
			}
			if err := c.Try(field.Type, tags); err != nil {
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

func (c *jsonCodec) Encode(val *reflect.Value) ([]byte, error) {
	return json.Marshal(val.Interface())
}

func (c *jsonCodec) Decode(data []byte, val *reflect.Value) error {
	return json.Unmarshal(data, val.Interface())
}

func init() {
	Register(&jsonCodec{})
}
