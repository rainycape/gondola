package input

import (
	"fmt"
	"gnd.la/i18n"
	"gnd.la/util/types"
	"math"
	"reflect"
	"strconv"
	"strings"
)

var (
	parserInterface = reflect.TypeOf((*Parser)(nil)).Elem()
)

// Parser is the interface implemented by types
// that know how to parse themselves from a user
// provided string.
type Parser interface {
	Parse(s string) error
}

// Parse tries to parse an string into the given argument.
// e.g.
//     var f float32
//     Parse("27.5", &f)
//     var width uint
//     Parse("57", &width)
// Supported types are: string, bool, u?int(8|16|32|64)? and float(32|64). If
// the parsed value would overflow the given type, the maximum value
// (or minimum, if it's negative) for the type will be set.
// If arg implements the Parser interface, its Parse method will
// be used instead.
func Parse(val string, arg interface{}) error {
	if parser, ok := arg.(Parser); ok {
		return parser.Parse(val)
	}
	v, err := types.SettableValue(arg)
	if err != nil {
		return err
	}
	return parse(val, v)
}

func parse(val string, v reflect.Value) error {
	var err error
	p := v
	// Get Pointer methods
	if p.IsValid() && p.Kind() != reflect.Ptr && p.CanAddr() {
		p = p.Addr()
	}
	if p.Type().Implements(parserInterface) {
		err = p.Interface().(Parser).Parse(val)
		if err == nil {
			return nil
		}
		// Continue with the function, just in case we can
		// still parse the value (e.g. an enum type which defines
		// a Parse() function for accepting raw strings but val
		// is actually a numeric value).
	}
	// If val is empty, set the value to zero
	if val == "" {
		v.Set(reflect.Zero(v.Type()))
		return nil
	}
	switch v.Type().Kind() {
	case reflect.Bool:
		res := false
		switch strings.ToLower(val) {
		case "", "f", "false", "0", "off":
		case "t", "true", "1", "on":
			res = true
		default:
			return i18n.Errorf("invalid boolean value %q", val)
		}
		v.SetBool(res)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		res, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return err
		}
		if v.OverflowInt(res) {
			if res > 0 {
				res = int64(math.Pow(2, float64(8*v.Type().Size()-1)) - 1)
			} else {
				res = -int64(math.Pow(2, float64(8*v.Type().Size()-1)))
			}
		}
		v.SetInt(res)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		res, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			return err
		}
		if v.OverflowUint(res) {
			res = uint64(math.Pow(2, float64(8*v.Type().Size())) - 1)
		}
		v.SetUint(res)
		return nil
	case reflect.Float32, reflect.Float64:
		res, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		v.SetFloat(res)
		return nil
	case reflect.String:
		v.SetString(val)
		return nil
	default:
		if err == nil {
			err = fmt.Errorf("Invalid argument type passed to Parse(): %s. Please, see the documentation for a list of the supported types.",
				v.Type())
		}
	}
	return err
}
