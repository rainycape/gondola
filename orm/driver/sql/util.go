package sql

import (
	"bytes"
	"reflect"

	"gnd.la/orm/index"
)

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Type().Kind() == reflect.Ptr && r.IsNil()
}

func DescField(idx *index.Index, field string) bool {
	val := idx.Get(index.DESC)
	switch x := val.(type) {
	case string:
		return x == field
	case []string:
		for _, v := range x {
			if v == field {
				return true
			}
		}
	case []interface{}:
		for _, v := range x {
			if s, ok := v.(string); ok && s == field {
				return true
			}
		}
	}
	return false
}

func buftos(buf *bytes.Buffer) string {
	return buf.String()
}
