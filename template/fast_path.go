package template

import (
	"html/template"
	"reflect"
)

// using reflect.Value.Call or reflect.Value.CallSlice is expensive because it
// validates all input values (which we already do, to replace invalid values with
// zeroes) and calculates the input and output sizes and alignments at runtime,
// for every call to the function. Also, it allocates an intermediate slice to pass
// the arguments and another slice for the return values. By defining our own functions,
// we can bypass a lot of those checks and reuse the input and output slices, since a
// *state won't be used by multiple goroutines at the same time.
// A fastPath function takes the form of:
//
//  func(in []reflect.Value, args []interface{}, out *reflect.Value) error
//
// Arguments have been validated to match the original input function. Non varargs are
// passed in "in", while variable arguments are in the "args" parameter. Fast path
// functions should replace their return value in the *out parameter and return the
// error, if any.

type fastPath func(in []reflect.Value, args []interface{}, out *reflect.Value) error

func newFastPath(f reflect.Value) fastPath {
	var fp fastPath
	switch x := f.Interface().(type) {
	case func(...interface{}) string:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(args...))
			return nil
		}
	case func(string, ...interface{}) string:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(in[0].String(), args...))
			return nil
		}
	case func(...interface{}) interface{}:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(args...))
			return nil
		}
	case func(...interface{}) (interface{}, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			val, err := x(args...)
			*out = reflect.ValueOf(val)
			return err
		}
	case func(interface{}, ...interface{}) (interface{}, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(in[0].Interface(), args...)
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}
	case func(...interface{}) (float64, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(args...)
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}
	case func(...interface{}) (int, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(args...)
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}
	case func(interface{}) interface{}:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(in[0].Interface()))
			return nil
		}
	case func(interface{}) (bool, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(in[0].Interface())
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}
	case func(interface{}) (int, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(in[0].Interface())
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}
	case func(...interface{}) bool:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(args...))
			return nil
		}
	case func(string) string:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(in[0].String()))
			return nil
		}
	case func(string) (string, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(in[0].String())
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}

	case func([]string, string) string:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			slice := in[0].Interface().([]string)
			*out = reflect.ValueOf(x(slice, in[1].String()))
			return nil
		}
	case func(string, ...interface{}) (string, error):
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			res, err := x(in[0].String(), args...)
			if err != nil {
				return err
			}
			*out = reflect.ValueOf(res)
			return nil
		}
	case func(string) template.HTML:
		fp = func(in []reflect.Value, args []interface{}, out *reflect.Value) error {
			*out = reflect.ValueOf(x(in[0].String()))
			return nil
		}
	}
	return fp
}
