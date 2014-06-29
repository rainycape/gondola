package template

// Functions available to gondola templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gnd.la/app/serialize"
	"gnd.la/html"
	"gnd.la/util/types"
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

func lt(arg1, arg2 interface{}) (bool, error) {
	v1 := reflect.ValueOf(arg1)
	v2 := reflect.ValueOf(arg2)
	t1 := v1.Type()
	t2 := v2.Type()
	switch {
	case types.IsInt(t1) && types.IsInt(t2):
		return v1.Int() < v2.Int(), nil
	case types.IsUint(t1) && types.IsUint(t2):
		return v1.Uint() < v2.Uint(), nil
	case types.IsFloat(t1) && types.IsFloat(t2):
		return v1.Float() < v2.Float(), nil
	}
	return false, fmt.Errorf("can't compare %T with %T", arg1, arg2)
}

func lte(arg1, arg2 interface{}) (bool, error) {
	lessThan, err := lt(arg1, arg2)
	if lessThan || err != nil {
		return lessThan, err
	}
	return eq(arg1, arg2), nil
}

func gt(arg1, arg2 interface{}) (bool, error) {
	v1 := reflect.ValueOf(arg1)
	v2 := reflect.ValueOf(arg2)
	t1 := v1.Type()
	t2 := v2.Type()
	switch {
	case types.IsInt(t1) && types.IsInt(t2):
		return v1.Int() > v2.Int(), nil
	case types.IsUint(t1) && types.IsUint(t2):
		return v1.Uint() > v2.Uint(), nil
	case types.IsFloat(t1) && types.IsFloat(t2):
		return v1.Float() > v2.Float(), nil
	}
	return false, fmt.Errorf("can't compare %T with %T", arg1, arg2)
}

func gte(arg1, arg2 interface{}) (bool, error) {
	greaterThan, err := gt(arg1, arg2)
	if greaterThan || err != nil {
		return greaterThan, err
	}
	return eq(arg1, arg2), nil
}

func jsons(arg interface{}) (string, error) {
	if jw, ok := arg.(serialize.JSONWriter); ok {
		var buf bytes.Buffer
		_, err := jw.WriteJSON(&buf)
		return buf.String(), err
	}
	b, err := json.Marshal(arg)
	return string(b), err
}

func _json(arg interface{}) (template.JS, error) {
	s, err := jsons(arg)
	return template.JS(s), err
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

func _map(args ...interface{}) (map[string]interface{}, error) {
	var key string
	m := make(map[string]interface{})
	for ii, v := range args {
		if ii%2 == 0 {
			if s, ok := v.(string); ok {
				key = s
			} else if s, ok := v.(*string); ok {
				key = *s
			} else {
				return nil, fmt.Errorf("invalid argument to map at index %d, %s instead of string", ii, reflect.TypeOf(v))
			}
		} else {
			m[key] = v
		}
	}
	return m, nil
}

// this returns *[]interface{} so append works on
// slices declared in templates
func _slice(args ...interface{}) *[]interface{} {
	// We need to copy the slice, since state.call
	// reuses a []interface{} to make all the calls
	// to variadic functions.
	ret := make([]interface{}, len(args))
	copy(ret, args)
	return &ret
}

func _append(items interface{}, args ...interface{}) (string, error) {
	val := reflect.ValueOf(items)
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return "", fmt.Errorf("first argument to append must be pointer to slice, it's %T", items)
	}
	sl := val.Elem()
	for _, v := range args {
		vval := reflect.ValueOf(v)
		if !vval.Type().AssignableTo(sl.Type().Elem()) {
			return "", fmt.Errorf("can't append %s to %s", vval.Type(), sl.Type())
		}
		sl = reflect.Append(sl, vval)
	}
	val.Elem().Set(sl)
	return "", nil
}

func deref(arg interface{}) (interface{}, error) {
	v := reflect.ValueOf(arg)
	if v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("argument to deref must be pointer, not %T", arg)
	}
	return v.Elem().Interface(), nil
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

func imult(args ...interface{}) (int, error) {
	val, err := mult(args...)
	return int(val), err
}

func add(args ...interface{}) (float64, error) {
	val := 0.0
	for ii, v := range args {
		value := reflect.ValueOf(v)
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val += float64(value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val += float64(value.Uint())
		case reflect.Float32, reflect.Float64:
			val += value.Float()
		case reflect.String:
			v, err := strconv.ParseFloat(value.String(), 64)
			if err != nil {
				return 0, fmt.Errorf("error parsing string passed to add() at index %d: %s", ii, err)
			}
			val += v
		default:
			return 0, fmt.Errorf("invalid argument of type %T passed to add() at index %d", v, ii)
		}
	}
	return val, nil
}

func iadd(args ...interface{}) (int, error) {
	val, err := add(args...)
	return int(val), err
}

func concat(args ...interface{}) string {
	s := make([]string, len(args))
	for ii, v := range args {
		s[ii] = types.ToString(v)
	}
	return strings.Join(s, "")
}

func and(args ...interface{}) interface{} {
	for _, v := range args {
		t, _ := types.IsTrue(v)
		if !t {
			return v
		}
	}
	return args[len(args)-1]
}

func or(args ...interface{}) interface{} {
	for _, v := range args {
		t, _ := types.IsTrue(v)
		if t {
			return v
		}
	}
	return args[len(args)-1]
}

func not(arg interface{}) bool {
	t, _ := types.IsTrue(arg)
	return !t
}

func divisible(n interface{}, d interface{}) (bool, error) {
	ni, err := types.ToInt(n)
	if err != nil {
		return false, fmt.Errorf("divisible() invalid number %v: %s", n, err)
	}
	di, err := types.ToInt(d)
	if err != nil {
		return false, fmt.Errorf("divisible() invalid divisor %v: %s", d, err)
	}
	return ni%di == 0, nil
}

func even(arg interface{}) (bool, error) {
	return divisible(arg, 2)
}

func odd(arg interface{}) (bool, error) {
	res, err := divisible(arg, 2)
	if err != nil {
		return false, err
	}
	return !res, nil
}

func now() time.Time {
	return time.Now()
}

func toHtml(s string) template.HTML {
	return template.HTML(strings.Replace(html.Escape(s), "\n", "<br>", -1))
}

func getVar(s *State, name string) interface{} {
	v, ok := s.Var(name)
	if !ok || !v.IsValid() {
		return nil
	}
	return v.Interface()
}

var templateFuncs = makeFuncMap(FuncMap{
	"#eq":        eq,
	"#neq":       neq,
	"#lt":        lt,
	"#lte":       lte,
	"#gt":        gt,
	"#gte":       gte,
	"#json":      _json,
	"#jsons":     jsons,
	"#nz":        nz,
	"#join":      strings.Join,
	"#map":       _map,
	"#slice":     _slice,
	"#append":    _append,
	"#deref":     deref,
	"#mult":      mult,
	"#imult":     imult,
	"#divisible": divisible,
	"#add":       add,
	"#iadd":      iadd,
	"#even":      even,
	"#odd":       odd,
	"#concat":    concat,
	"#and":       and,
	"#or":        or,
	"#not":       not,
	"now":        now,
	"#split":     strings.Split,
	"#split_n":   strings.SplitN,
	"#to_lower":  strings.ToLower,
	"#to_title":  strings.ToTitle,
	"#to_upper":  strings.ToUpper,
	"#to_html":   toHtml,

	// state manipulation functions
	"@var": getVar,

	// Go builtins
	"call":      call,
	"#html":     template.HTMLEscaper,
	"#index":    index,
	"#js":       template.JSEscaper,
	"#len":      length,
	"#print":    fmt.Sprint,
	"#printf":   fmt.Sprintf,
	"#println":  fmt.Sprintln,
	"#urlquery": template.URLQueryEscaper,

	// Pseudo-functions which act as custom tags
	"extend": nop,
	// Used to make the parser parse undefined
	// variables, since we allow variable
	// inheritance to subtemplates
	varNop: nop,
})
