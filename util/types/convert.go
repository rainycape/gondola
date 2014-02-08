package types

import (
	"fmt"
	"reflect"
)

// Convert converts the in parameter to the out parameter, which
// must be a non-nil pointer, usually to a different type than in. Supported
// parameters type are u?int(8|16|32|64)?, float(32|64) and string.
func Convert(in interface{}, out interface{}) error {
	ov := reflect.ValueOf(out)
	if !ov.IsValid() || ov.Kind() != reflect.Ptr || ov.IsNil() {
		return fmt.Errorf("can't convert into %T - must be non-nil pointer", out)
	}
	oe := ov.Elem()
	otyp := oe.Type()
	switch {
	case IsInt(otyp):
		i, err := ToInt(in)
		if err != nil {
			return err
		}
		oe.Set(reflect.ValueOf(i))
	case IsUint(otyp):
		u, err := ToUint(in)
		if err != nil {
			return err
		}
		oe.Set(reflect.ValueOf(u))
	case IsFloat(otyp):
		f, err := ToFloat(in)
		if err != nil {
			return err
		}
		oe.Set(reflect.ValueOf(f))
	case otyp.Kind() == reflect.String:
		if b, ok := in.([]byte); ok {
			oe.Set(reflect.ValueOf(string(b)))
		} else {
			oe.Set(reflect.ValueOf(ToString(in)))
		}
	default:
		return fmt.Errorf("can't convert from %T to %T", in, out)
	}
	return nil
}
