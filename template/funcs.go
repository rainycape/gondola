package template

// Functions available to gondola templates

import (
	"encoding/json"
	"fmt"
	"gondola/files"
	"gondola/template/config"
	"html/template"
	"net/http"
	"reflect"
	"strings"
)

func asset(name ...string) string {
	n := strings.Join(name, "")
	return files.StaticFileUrl(config.StaticFilesUrl(), n)
}

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
		if x != 0 {
			return true
		}
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

var templateFuncs template.FuncMap = template.FuncMap{
	"asset": asset,
	"eq":    eq,
	"neq":   neq,
	"json":  _json,
	"nz":    nz,
	"lower": lower,
	"join":  join,
}

// Other functions which are defined depending on the template
// (because they require access to the context or the mux)
// Ctx
// reverse
// request

func makeContext(t *Template) func() interface{} {
	return func() interface{} {
		return t.context
	}
}

func makeReverse(t *Template) func(string, ...interface{}) string {
	return func(name string, args ...interface{}) string {
		val, err := t.context.Mux().Reverse(name, args...)
		if err != nil {
			panic(err)
		}
		return val
	}
}

func makeRequest(t *Template) func() *http.Request {
	return func() *http.Request {
		return t.context.R
	}
}
