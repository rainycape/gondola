package generic

import (
	"fmt"
	"reflect"
)

// Keys returns the keys of the map m as a slice. Given a map m of type
// map[T1]T2 the return value will be of type []T1. If m is not a map,
// Keys will panic.
func Keys(m interface{}) interface{} {
	mval := reflect.ValueOf(m)
	mt := mval.Type()
	if mt.Kind() != reflect.Map {
		panic(fmt.Errorf("argument to Keys() must be map, not %T", m))
	}
	count := mval.Len()
	out := reflect.MakeSlice(reflect.SliceOf(mt.Key()), count, count)
	for ii, v := range mval.MapKeys() {
		out.Index(ii).Set(v)
	}
	return out.Interface()
}

// Values returns the values of the map m as a slice. Given a map m of type
// map[T1]T2 the return value will be of type []T2. If m is not a map,
// Values will panic.
func Values(m interface{}) interface{} {
	mval := reflect.ValueOf(m)
	mt := mval.Type()
	if mt.Kind() != reflect.Map {
		panic(fmt.Errorf("argument to Values() must be map, not %T", m))
	}
	count := mval.Len()
	out := reflect.MakeSlice(reflect.SliceOf(mt.Elem()), count, count)
	for ii, v := range mval.MapKeys() {
		out.Index(ii).Set(mval.MapIndex(v))
	}
	return out.Interface()
}
