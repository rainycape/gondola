package runtimeutil

import (
	"reflect"
	"runtime"
)

// FuncName returns the name for the given function.
// If f is not a function or its name can't be determined
// an empty string is returned.
func FuncName(f interface{}) string {
	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Func {
		if fn := runtime.FuncForPC(v.Pointer()); fn != nil {
			return fn.Name()
		}
	}
	return ""
}
