package generic

import (
	"fmt"
	"reflect"
)

// Select returns a new slice with the selected key
// extracted from the sl argument. The key must be either
// a field name or a method name with no arguments and just
// one return value. The returned value is a slice of the same
// type of the selected field.
// e.g.
//
//  type Person struct {
//	Name string
//  }
//  ...
//  var persons []*Person = ...
//  names := Select(persons, "Name").([]string)
//
// This function is around 2 times slower than the specific
// code for extracting the field.
func Select(sl interface{}, key string) interface{} {
	fn, val, elem, typ, err := sliceMapper(sl, key)
	if err != nil {
		panic(err)
	}
	count := val.Len()
	src := getHandle(val)
	out := reflect.MakeSlice(reflect.SliceOf(typ), count, count)
	dst := getHandle(out)
	idx := indexer(elem)
	set := indexSetter(typ)
	for ii := 0; ii < count; ii++ {
		e := idx(src, ii)
		set(dst, ii, fn(e))
	}
	return out.Interface()
}

// Contains returns wheter the given slice or array contains
// the given value. Given a slice of type []T, val must be
// of type T. Otherwise, this function will panic.
func Contains(iterable interface{}, val interface{}) bool {
	itr := reflect.ValueOf(iterable)
	if itr.Kind() != reflect.Slice && itr.Kind() != reflect.Array {
		panic(fmt.Errorf("first argument to Contains must be slice or array, not %T", iterable))
	}
	v := reflect.ValueOf(val)
	if itr.Type().Elem() != v.Type() {
		panic(fmt.Errorf("second argument to Contains must be %s, not %s", itr.Type().Elem(), v.Type()))
	}
	vi := v.Interface()
	for ii := 0; ii < itr.Len(); ii++ {
		if reflect.DeepEqual(itr.Index(ii).Interface(), vi) {
			return true
		}
	}
	return false
}
