package template

// Functions available to gondola templates

import (
	"encoding/json"
	"fmt"
	"gondola/assets"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	"time"
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

func mult(args ...interface{}) (float64, error) {
	val := 1.0
	for ii, v := range args {
		value := reflect.ValueOf(v)
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val *= float64(value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val *= float64(value.Uint())
		case reflect.Float32, reflect.Float64:
			val *= value.Float()
		case reflect.String:
			v, err := strconv.ParseFloat(value.String(), 64)
			if err != nil {
				return 0, fmt.Errorf("Error parsing string passed to mult at index %d: %s", ii, err)
			}
			val *= v
		default:
			return 0, fmt.Errorf("Invalid argument of type %T passed to mult at index %d", v, ii)
		}
	}
	return val, nil

}

func concat(args ...interface{}) string {
	var s []string
	for _, v := range args {
		s = append(s, fmt.Sprintf("%v", v))
	}
	return strings.Join(s, "")
}

func and(args ...interface{}) bool {
	for _, v := range args {
		val := reflect.ValueOf(v)
		t, _ := isTrue(val)
		if !t {
			return false
		}
	}
	return true
}

func or(args ...interface{}) bool {
	for _, v := range args {
		val := reflect.ValueOf(v)
		t, _ := isTrue(val)
		if t {
			return true
		}
	}
	return false
}

func not(arg interface{}) bool {
	val := reflect.ValueOf(arg)
	t, _ := isTrue(val)
	return !t
}

func now() time.Time {
	return time.Now()
}

// isTrue returns whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value. This function is a copy of the
// one found in text/template
func isTrue(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return false, true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true // Struct values are always true.
	default:
		return
	}
	return truth, true
}

var templateFuncs template.FuncMap = template.FuncMap{
	"eq":     eq,
	"neq":    neq,
	"json":   _json,
	"nz":     nz,
	"lower":  lower,
	"join":   join,
	"map":    _map,
	"mult":   mult,
	"render": assets.Render,
	"concat": concat,
	"and":    and,
	"or":     or,
	"not":    not,
	"now":    now,
}
