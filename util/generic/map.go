package generic

import (
	"fmt"
	"reflect"
)

// Keys returns the keys of the map m in the pointer to slice out.
// The key type of the map must match the element type of the slice.
// Any type mismatch will result in a panic, since it's a programming
// error.
func Keys(m interface{}, out interface{}) {
	if err := keys(m, out); err != nil {
		panic(err)
	}
}

func keys(m interface{}, out interface{}) error {
	mval := reflect.ValueOf(m)
	mt := mval.Type()
	if mt.Kind() != reflect.Map {
		return fmt.Errorf("first argument to Keys() must be map, not %T", m)
	}
	oval := reflect.ValueOf(out)
	ot := oval.Type()
	if ot.Kind() != reflect.Ptr || ot.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("second argument to Keys() must be pointer to slice, not %T", out)
	}
	if mt.Key() != ot.Elem().Elem() {
		return fmt.Errorf("map key type and slice element type must match, but instead they're %s and %s", mt.Key(), ot.Elem().Elem())
	}
	count := mval.Len()
	ks := reflect.MakeSlice(oval.Type().Elem(), count, count)
	for ii, v := range mval.MapKeys() {
		ks.Index(ii).Set(v)
	}
	oval.Elem().Set(ks)
	return nil
}

// Values returns the values of the map m in the pointer to slice out.
// The value type of the map must match the element type of the slice.
// Any type mismatch will result in a panic, since it's a programming
// error.
func Values(m interface{}, out interface{}) {
	if err := values(m, out); err != nil {
		panic(err)
	}
}

func values(m interface{}, out interface{}) error {
	mval := reflect.ValueOf(m)
	mt := mval.Type()
	if mt.Kind() != reflect.Map {
		return fmt.Errorf("first argument to Values() must be map, not %T", m)
	}
	oval := reflect.ValueOf(out)
	ot := oval.Type()
	if ot.Kind() != reflect.Ptr || ot.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("second argument to Values() must be pointer to slice, not %T", out)
	}
	if mt.Elem() != ot.Elem().Elem() {
		return fmt.Errorf("map element type and slice element type must match, but instead they're %s and %s", mt.Elem(), ot.Elem().Elem())
	}
	count := mval.Len()
	ks := reflect.MakeSlice(oval.Type().Elem(), count, count)
	for ii, v := range mval.MapKeys() {
		ks.Index(ii).Set(mval.MapIndex(v))
	}
	oval.Elem().Set(ks)
	return nil
}
