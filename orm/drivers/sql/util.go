package sql

import (
	"gondola/orm/driver"
	"reflect"
)

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Type().Kind() == reflect.Ptr && r.IsNil()
}

func DescField(idx driver.Index, field string) bool {
	if strs, ok := idx.Get(driver.DESC).([]string); ok {
		for _, v := range strs {
			if v == field {
				return true
			}
		}
	}
	return false
}
