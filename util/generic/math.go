package generic

import (
	"fmt"
	"reflect"
)

func minOrMax(in interface{}, pickMin bool) interface{} {
	inval := iterable(in)
	t := inval.Type().Elem()
	cmp := getReflectComparator(t)
	if cmp == nil {
		panic(fmt.Errorf("can't compare type %s", t))
	}
	out := reflect.New(t).Elem()
	if inval.Len() == 0 {
		return out.Interface()
	}
	out.Set(inval.Index(0))
	if pickMin {
		for ii := 1; ii < inval.Len(); ii++ {
			cur := inval.Index(ii)
			if cmp(&cur, &out) {
				out.Set(cur)
			}
		}
	} else {
		for ii := 1; ii < inval.Len(); ii++ {
			cur := inval.Index(ii)
			if !cmp(&cur, &out) {
				out.Set(cur)
			}
		}
	}
	return out.Interface()
}

// Min returns the smallest element from the in argument, which
// must be a slice or array of a comparable type. Comparable types
// include:
//
//  - basic types with a defined < operator
//  - slices (their length is compared)
//
// If the element type is not comparable, Min will panic.
// Given a slice of type []T, the returned value will be
// of type T.
func Min(in interface{}) interface{} {
	return minOrMax(in, true)
}

// Max returns the biggest element from the in argument, which
// must be a slice or array of a comparable type. Comparable types
// include:
//
//  - basic types with a defined < operator
//  - slices (their length is compared)
//
// If the element type is not comparable, Max will panic.
// Given a slice of type []T, the returned value will be
// of type T.
func Max(in interface{}) interface{} {
	return minOrMax(in, false)
}

// Sum returns the sum of every element from the in argument, which
// must be a slice or array of a summable type. Summable types
// include:
//
//  - basic types with a defined + operator
//  - slices (they are concatenated)
//
// If the element type is not summable, Sum will panic.
// Given a slice of type []T, the returned value will be
// of type T.
func Sum(in interface{}) interface{} {
	inval := iterable(in)
	t := inval.Type().Elem()
	add := getReflectAdder(t)
	if add == nil {
		panic(fmt.Errorf("can't sum type %s", t))
	}
	out := reflect.New(t).Elem()
	for ii := 0; ii < inval.Len(); ii++ {
		cur := inval.Index(ii)
		out = add(&out, &cur)
	}
	return out.Convert(t).Interface()
}

// Avg returns the average of the elements from the in argument, which
// must be a slice or array of a summable and divisible type. Summable types
// include:
//
//  - basic types with a defined + operator and a defined / operator
//
// If the element type is not summable and divisible, Avg will panic.
// Given a slice of type []T, the returned value will be
// of type T.
func Avg(in interface{}) interface{} {
	res := Sum(in)
	s := reflect.ValueOf(in).Len()
	if s == 0 {
		return res
	}
	v := reflect.ValueOf(res)
	var avg interface{}
	switch v.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		avg = v.Int() / int64(s)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		avg = v.Uint() / uint64(s)
	case reflect.Float32, reflect.Float64:
		avg = v.Float() / float64(s)
	default:
		panic(fmt.Errorf("can't calculate average for type %T", in))
	}
	return reflect.ValueOf(avg).Convert(v.Type()).Interface()
}
