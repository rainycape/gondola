package generic

import (
	"fmt"
	"reflect"
)

func iterable(in interface{}) reflect.Value {
	v := reflect.ValueOf(in)
	if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
		panic(fmt.Errorf("first argument must be array or slice, not %T", in))
	}
	return v
}

// Filter applies the predicate function f over the elements in the first
// argument, returning a new slice of the same type with the elements for
// which f returns true. Given a slice of type T, f must be of the form:
//
//  func(T) bool
//
// If in is not a slice or array or f has not the right type, Filter will panic.
func Filter(in interface{}, f interface{}) interface{} {
	inval := iterable(in)
	fval := reflect.ValueOf(f)
	if !fval.IsValid() || fval.Kind() != reflect.Func ||
		fval.Type().NumIn() != 1 || fval.Type().NumOut() != 1 ||
		fval.Type().In(0) != inval.Type().Elem() || fval.Type().Out(0) != boolType {
		panic(fmt.Errorf("second argument must be func(%s) %s, not %T", inval.Type().Elem(), boolType, f))
	}
	rin := make([]reflect.Value, 1)
	out := reflect.MakeSlice(inval.Type(), 0, 0)
	for ii := 0; ii < inval.Len(); ii++ {
		rin[0] = inval.Index(ii)
		res := fval.Call(rin)
		if res[0].Bool() {
			out = reflect.Append(out, rin[0])
		}
	}
	return out.Interface()
}

// Map returns a new slice of the same length as the first argument, where
// the elements are the result of applying f to each element in the input.
// The returned slice has the element type that f returns. Given a slice
// of type T1, f must be of the form:
//
//  func(T1) T2 // note that T2 can be equal to T1
//
// The returned slice will be of type []T2.
//
// If in is not a slice or array or f has not the right type, Map will panic.
func Map(in interface{}, f interface{}) interface{} {
	inval := iterable(in)
	fval := reflect.ValueOf(f)
	if !fval.IsValid() || fval.Kind() != reflect.Func ||
		fval.Type().NumIn() != 1 || fval.Type().NumOut() != 1 ||
		fval.Type().In(0) != inval.Type().Elem() {
		panic(fmt.Errorf("second argument must be func(%s) T, not %T", inval.Type().Elem(), f))
	}
	rin := make([]reflect.Value, 1)
	out := reflect.MakeSlice(reflect.SliceOf(fval.Type().Out(0)), inval.Len(), inval.Len())
	for ii := 0; ii < inval.Len(); ii++ {
		rin[0] = inval.Index(ii)
		res := fval.Call(rin)
		out.Index(ii).Set(res[0])
	}
	return out.Interface()
}

// Reduce applies f cumulatively to the elements in the first argument.
// The second argument is used as the initial value and should usually be
// the neutral element for the binary operation represented by f (e.g.
// 0 for addition, 1 for multiplication, etc...
// Given a slice of type T, the following conditions must be satisfied:
//
//  f -> func(T, T) T
//  start -> assignable to T
//
// If in is not an array or slice or the previous conditions are not
// satisfied, Reduce will panic.
func Reduce(in interface{}, start interface{}, f interface{}) interface{} {
	inval := iterable(in)
	fval := reflect.ValueOf(f)
	t := inval.Type().Elem()
	if !fval.IsValid() || fval.Kind() != reflect.Func ||
		fval.Type().NumIn() != 2 || fval.Type().NumOut() != 1 ||
		fval.Type().In(0) != t || fval.Type().In(1) != t ||
		fval.Type().Out(0) != t {
		return fmt.Errorf("third argument must be func(%s, %s) %s, not %T", t, t, t, f)
	}
	sval := reflect.ValueOf(start)
	out := reflect.New(t).Elem()
	if !sval.IsValid() {
		panic(fmt.Errorf("second argument of type %T is invalid", start))
	}
	if !sval.Type().AssignableTo(out.Type()) {
		if !sval.Type().ConvertibleTo(out.Type()) {
			panic(fmt.Errorf("second argument must be convertible to %s, not %T", out.Type(), start))
		}
		sval = sval.Convert(out.Type())
	}
	out.Set(sval)
	rin := make([]reflect.Value, 2)
	for ii := 0; ii < inval.Len(); ii++ {
		rin[0] = out
		rin[1] = inval.Index(ii)
		res := fval.Call(rin)
		out.Set(res[0])
	}
	return out.Interface()
}
