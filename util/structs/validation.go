package structs

import (
	"fmt"
	"reflect"
)

// TODO: Make these functions work with nested fields and embedded structs

func values(args []interface{}) []reflect.Value {
	ret := make([]reflect.Value, len(args))
	for ii, v := range args {
		ret[ii] = reflect.ValueOf(v)
	}
	return ret
}

// ValidationFunction returns the validation function in obj to validate
// the field argument with the given arguments. See the documentation on
// Validate to learn more about validation functions.
func ValidationFunction(obj interface{}, field string, args ...interface{}) (fn reflect.Value, err error) {
	val := reflect.ValueOf(obj)
	pval := val
	// Get pointer methods
	if pval.CanAddr() {
		pval = pval.Addr()
	}
	fname := fmt.Sprintf("Validate%s", field)
	f := pval.MethodByName(fname)
	if !f.IsValid() {
		eval := val
		for eval.Type().Kind() == reflect.Ptr {
			eval = eval.Elem()
		}
		ff := eval.FieldByName(fname)
		if ff.IsValid() && ff.Type().Kind() == reflect.Func {
			f = ff
		}
	}
	if f.IsValid() {
		// Check validation function arguments and return value
		ft := f.Type()
		if ft.NumOut() != 1 || ft.Out(0).String() != "error" {
			err = fmt.Errorf("invalid validation function %s (type %s): it must return just one value of type error", fname, val.Type)
			return
		}
		if ft.NumIn() > 0 {
			vals := values(args)
			for ii, v := range vals {
				if ii < ft.NumIn() && v.Type() != ft.In(ii) {
					err = fmt.Errorf("invalid validation function %s (type %s): argument #%d must be of type %s", fname, val.Type, ii, v.Type)
					return
				}
			}
		}
		return f, nil
	}
	return
}

// Validate calls the validation function in obj for the given field, if any.
// Given a field F, its validation function must be called ValidateF. It might
// be either a method or a field named like that. If both exist, the method
// takes precedence. Validation functions must return a single argument of
// type error. As for the input parameters of the validation function, their
// types must match the types of the args parameter. However, validation
// functions are allowed to have a different number of arguments than the
// amount provided in args. If the validation function takes less arguments
// any additional arguments are ignored. On the other hand, if the validation
// function receives more arguments that len(args), they will be zero for
// the corresponding type (as returned by reflect.Zero).
//
// Validate returns an error in the following three cases:
//  - The function exists, but its return type is not error (or has a number of return values != 1).
//  - The function exists, but its arguments don't match the types in args.
//  - The function exists and when executed returns an error != nil.
func Validate(obj interface{}, field string, args ...interface{}) error {
	fn, err := ValidationFunction(obj, field, args...)
	if err != nil {
		return err
	}
	if fn.IsValid() {
		in := values(args)
		fin := fn.Type().NumIn()
		if len(in) > fin {
			in = in[:fin]
		} else if fin > len(in) {
			for ii := len(in); ii < fin; ii++ {
				in = append(in, reflect.Zero(fn.Type().In(ii)))
			}
		}
		res := fn.Call(in)
		if len(res) > 0 && !res[0].IsNil() {
			return res[0].Interface().(error)
		}
	}
	return nil
}
