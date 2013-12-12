package generic

import (
	"fmt"
	"reflect"
)

// Select returns a new slice with the selected key
// extracted from the sl argument. The key must be either
// a field name or a method name with no arguments and just
// one return value. The out argument must be a pointer to
// a slice of the same type yielded by extracting the key.
// e.g.
//
//  type Person struct {
//	Name string
//  }
//  ...
//  var persons []*Person = ...
//  var names []string
//  Select(persons, "Name", &names)
//
// This function is around 2 times slower than the specific
// code for extracting the field.
func Select(sl interface{}, key string, out interface{}) {
	oval := reflect.ValueOf(out)
	if oval.Kind() != reflect.Ptr || oval.Elem().Kind() != reflect.Slice {
		panic(fmt.Errorf("out argument to Select() must be pointer to slice, not %T", out))
	}
	ot := oval.Type().Elem()
	oelem := ot.Elem()
	fn, val, elem, typ, err := sliceMapper(sl, key)
	if err != nil {
		panic(err)
	}
	if oelem != typ {
		panic(fmt.Errorf("key %q yields type %s, but output slice is of type %s", typ, oelem))
	}
	count := val.Len()
	src := getHandle(val)
	osl := oval.Elem()
	if osl.IsValid() && osl.Cap() >= count {
		osl.SetCap(count)
		osl.SetLen(count)
	} else {
		osl = reflect.MakeSlice(ot, count, count)
		oval.Elem().Set(osl)
	}
	dst := getHandle(osl)
	idx := indexer(elem)
	set := indexSetter(typ)
	for ii := 0; ii < count; ii++ {
		e := idx(src, ii)
		set(dst, ii, fn(e))
	}
}
