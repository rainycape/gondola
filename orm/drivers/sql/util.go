package sql

import (
	"gondola/orm/index"
	"reflect"
)

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Type().Kind() == reflect.Ptr && r.IsNil()
}

func DescField(idx *index.Index, field string) bool {
	if strs, ok := idx.Get(index.DESC).([]string); ok {
		for _, v := range strs {
			if v == field {
				return true
			}
		}
	}
	return false
}
