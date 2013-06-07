package sql

import (
	"reflect"
)

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Type().Kind() == reflect.Ptr && r.IsNil()
}
