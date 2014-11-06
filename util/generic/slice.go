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

// Remove removes the first occurence of the given val. It returns true
// if an element was removed, false otherwise. Note that the first argument
// must a pointer to a slice, while the second one must be a valid element
// of the slice. Otherwise, this function will panic.
func Remove(slicePtr interface{}, val interface{}) bool {
	s := reflect.ValueOf(slicePtr)
	if s.Kind() != reflect.Ptr || s.Type().Elem().Kind() != reflect.Slice {
		panic(fmt.Errorf("first argument to Remove must be pointer to slice, not %T", slicePtr))
	}
	if s.IsNil() {
		return false
	}
	itr := s.Elem()
	v := reflect.ValueOf(val)
	if itr.Type().Elem() != v.Type() {
		panic(fmt.Errorf("second argument to Remove must be %s, not %s", itr.Type().Elem(), v.Type()))
	}
	vi := v.Interface()
	for ii := 0; ii < itr.Len(); ii++ {
		if reflect.DeepEqual(itr.Index(ii).Interface(), vi) {
			newSlice := reflect.MakeSlice(itr.Type(), itr.Len()-1, itr.Len()-1)
			reflect.Copy(newSlice, itr.Slice(0, ii))
			reflect.Copy(newSlice.Slice(ii, newSlice.Len()), itr.Slice(ii+1, itr.Len()))
			s.Elem().Set(newSlice)
			return true
		}
	}
	return false
}

func byOutputMap(iterable interface{}, field string, fn string, slices bool) (itr reflect.Value, f reflect.StructField, out reflect.Value, err error) {
	itr = reflect.ValueOf(iterable)
	if itr.Kind() != reflect.Slice && itr.Kind() != reflect.Array {
		err = fmt.Errorf("first argument to %s must be slice or array, not %T", fn, iterable)
		return
	}
	elem := itr.Type().Elem()
	s := elem
	for s.Kind() == reflect.Ptr {
		s = s.Elem()
	}
	if s.Kind() != reflect.Struct {
		err = fmt.Errorf("first argument to %s must contain structs or pointers to structs, not %v", fn, elem)
		return
	}
	var ok bool
	f, ok = s.FieldByName(field)
	if !ok {
		err = fmt.Errorf("type %v does not have a field named %q", elem, field)
		return
	}
	if slices {
		elem = reflect.SliceOf(elem)
	}
	mapType := reflect.MapOf(f.Type, elem)
	out = reflect.MakeMap(mapType)
	return
}

// By returns a map with the elements of the given iterable, which must be
// a slice of structs or pointers to structs, where the key is the field
// named by the field argument. If there are multiple elements with the same
// field value, only the last one will be in the returned map. If you want
// multiple elements to be returned use GroupsBy. Given a slice of type []T
// with a field of type int this function will return a map[int]T. If the
// arguments are not valid, this function will panic.
func By(iterable interface{}, field string) interface{} {
	itr, f, out, err := byOutputMap(iterable, field, "By", false)
	if err != nil {
		panic(err)
	}
elements:
	for ii := 0; ii < itr.Len(); ii++ {
		v := itr.Index(ii)
		val := v
		for val.Kind() == reflect.Ptr {
			if val.IsNil() {
				continue elements
			}
			val = val.Elem()
		}
		fieldValue := val.FieldByIndex(f.Index)
		out.SetMapIndex(fieldValue, v)
	}
	return out.Interface()
}

// GroupsBy returns a map with slices of elements from the given iterable,
// which must be a slice of structs or pointers to structs, where the key is
// the field named by the field argument. Map values will be slices of the
// elements with the same key. If you only want an element per key, use By
// instead. Given a slice of type []T with a field of type int this function
// will return a map[int][]T. If the arguments are not valid, this function
// will panic.
func GroupsBy(iterable interface{}, field string) interface{} {
	itr, f, out, err := byOutputMap(iterable, field, "GroupsBy", true)
	if err != nil {
		panic(err)
	}
	sliceType := out.Type().Elem()
elements:
	for ii := 0; ii < itr.Len(); ii++ {
		v := itr.Index(ii)
		val := v
		for val.Kind() == reflect.Ptr {
			if val.IsNil() {
				continue elements
			}
			val = val.Elem()
		}
		fieldValue := val.FieldByIndex(f.Index)
		slice := out.MapIndex(fieldValue)
		if !slice.IsValid() {
			slice = reflect.MakeSlice(sliceType, 0, 1)
		}
		slice = reflect.Append(slice, v)
		out.SetMapIndex(fieldValue, slice)
	}
	return out.Interface()
}
