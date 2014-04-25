package orm

import (
	"reflect"
	"time"
)

var (
	ormFuncs = map[string]reflect.Value{
		"now":   reflect.ValueOf(funcNow),
		"today": reflect.ValueOf(funcToday),
	}
)

func funcNow() time.Time {
	return time.Now().UTC()
}

func funcToday() time.Time {
	now := funcNow()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
