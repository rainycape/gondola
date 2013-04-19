package template

// Functions available to gondola templates

import (
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"strings"
)

func eq(args ...interface{}) bool {
	if len(args) == 0 {
		return false
	}
	x := args[0]
	switch x := x.(type) {
	case string, int, int64, byte, float32, float64:
		for _, y := range args[1:] {
			if x == y {
				return true
			}
		}
		return false
	}

	for _, y := range args[1:] {
		if reflect.DeepEqual(x, y) {
			return true
		}
	}
	return false
}

func neq(args ...interface{}) bool {
	return !eq(args...)
}

func _json(arg interface{}) string {
	if arg == nil {
		return ""
	}
	b, err := json.Marshal(arg)
	if err == nil {
		return string(b)
	}
	return ""
}

func nz(x interface{}) bool {
	switch x := x.(type) {
	case int, uint, int64, uint64, byte, float32, float64:
		return x != 0
	case string:
		return len(x) > 0
	}
	return false
}

func lower(x string) string {
	return strings.ToLower(x)
}

func join(x []string, sep string) string {
	s := ""
	for _, v := range x {
		s += fmt.Sprintf("%v%s", v, sep)
	}
	if len(s) > 0 {
		return s[:len(s)-len(sep)]
	}
	return ""
}

func _map(args ...interface{}) (map[string]interface{}, error) {
	var key string
	m := make(map[string]interface{})
	for ii, v := range args {
		if ii%2 == 0 {
			if s, ok := v.(string); ok {
				key = s
			} else {
				return nil, fmt.Errorf("Invalid argument to map at index %d, %t instead of string", ii, v)
			}
		} else {
			m[key] = v
		}
	}
	return m, nil
}

var templateFuncs template.FuncMap = template.FuncMap{
	"eq":    eq,
	"neq":   neq,
	"json":  _json,
	"nz":    nz,
	"lower": lower,
	"join":  join,
	"map":   _map,
}
