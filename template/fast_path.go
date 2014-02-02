package template

import (
	"reflect"
)

type fastPath func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error

func newFastPath(f reflect.Value) fastPath {
	var fp fastPath
	switch x := f.Interface().(type) {
	case func(...interface{}) string:
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			v := reflect.ValueOf(x(args...))
			*out = append(*out, v)
			return nil
		}
	case func(string, ...interface{}) string:
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			v := reflect.ValueOf(x(in[0].String(), args...))
			*out = append(*out, v)
			return nil
		}
	case func(...interface{}) interface{}:
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			v := reflect.ValueOf(x(args...))
			*out = append(*out, v)
			return nil
		}
	case func(interface{}, ...interface{}) (interface{}, error):
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			res, err := x(in[0].Interface(), args...)
			if err != nil {
				return err
			}
			*out = append(*out, reflect.ValueOf(res))
			return nil
		}
	case func(...interface{}) (float64, error):
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			res, err := x(args...)
			if err != nil {
				return err
			}
			*out = append(*out, reflect.ValueOf(res))
			return nil
		}
	case func(...interface{}) (int, error):
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			res, err := x(args...)
			if err != nil {
				return err
			}
			*out = append(*out, reflect.ValueOf(res))
			return nil
		}
	case func(...interface{}) bool:
		fp = func(in []reflect.Value, args []interface{}, out *[]reflect.Value) error {
			v := reflect.ValueOf(x(args...))
			*out = append(*out, v)
			return nil
		}
	}
	return fp
}
