package types

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

var (
	ErrCantSet = errors.New("can't set value (you might need to pass a pointer)")
)

func SettableValue(val interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(val)
	for v.Type().Kind() == reflect.Ptr {
		if !v.Elem().IsValid() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if !v.CanSet() {
		return reflect.Value{}, ErrCantSet
	}
	return v, nil
}

// IsNumeric returns true iff the type is one of
// the int, uint, float or complex types.
func IsNumeric(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Int || k == reflect.Uint ||
		k == reflect.Float64 || k == reflect.Float32 ||
		k == reflect.Int8 || k == reflect.Uint8 ||
		k == reflect.Int16 || k == reflect.Uint16 ||
		k == reflect.Int32 || k == reflect.Uint32 ||
		k == reflect.Int64 || k == reflect.Uint64 ||
		k == reflect.Complex128 || k == reflect.Complex64
}

type rvs []reflect.Value

func (x rvs) Len() int      { return len(x) }
func (x rvs) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

type rvInts struct{ rvs }

func (x rvInts) Less(i, j int) bool { return x.rvs[i].Int() < x.rvs[j].Int() }

type rvUints struct{ rvs }

func (x rvUints) Less(i, j int) bool { return x.rvs[i].Uint() < x.rvs[j].Uint() }

type rvFloats struct{ rvs }

func (x rvFloats) Less(i, j int) bool { return x.rvs[i].Float() < x.rvs[j].Float() }

type rvStrings struct{ rvs }

func (x rvStrings) Less(i, j int) bool { return x.rvs[i].String() < x.rvs[j].String() }

// SortValues sorts the given slice of reflect.Value, if
// possible. If the slice can't be sorted, an error is
// returned.
func SortValues(v []reflect.Value) error {
	if len(v) <= 1 {
		return nil
	}
	switch v[0].Kind() {
	case reflect.Float32, reflect.Float64:
		sort.Sort(rvFloats{v})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sort.Sort(rvInts{v})
	case reflect.String:
		sort.Sort(rvStrings{v})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		sort.Sort(rvUints{v})
	default:
		return fmt.Errorf("can't sort %T", v[0].Interface())
	}
	return nil
}
