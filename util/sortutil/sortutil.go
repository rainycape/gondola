// Package sortutil contains functions for easily
// sorting slices while avoiding a lot of boilerplate.
//
// Keep in mind though, that this convenience has a significant
// runtime penalty, so you shouldn't use it for long lists.
package sortutil

import (
	"fmt"
	"reflect"
	"sort"
)

type sortable struct {
	value      reflect.Value
	key        func(reflect.Value) reflect.Value
	descending bool
}

func fieldValue(key string) func(reflect.Value) reflect.Value {
	return func(v reflect.Value) reflect.Value {
		return reflect.Indirect(v).FieldByName(key)
	}
}

func methodValue(key string) func(reflect.Value) reflect.Value {
	return func(v reflect.Value) reflect.Value {
		return v.MethodByName(key).Call(nil)[0]
	}
}

func (s *sortable) Len() int {
	return s.value.Len()
}

func (s *sortable) less(i, j int) bool {
	fi := s.key(s.value.Index(i))
	fj := s.key(s.value.Index(j))
	switch fi.Kind() {
	case reflect.Bool:
		return !fi.Bool() && fj.Bool()
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return fi.Int() < fj.Int()
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return fi.Uint() < fj.Uint()
	case reflect.Float32, reflect.Float64:
		return fi.Float() < fj.Float()
	case reflect.String:
		return fi.String() < fj.String()
	default:
		panic(fmt.Errorf("can't compare type %s", fi.Type()))
	}
	panic("unreachable")
}

func (s *sortable) Less(i, j int) bool {
	v := s.less(i, j)
	if s.descending {
		return !v
	}
	return v
}

func (s *sortable) Swap(i, j int) {
	vi := s.value.Index(i)
	vj := s.value.Index(j)
	tmp := reflect.New(vi.Type()).Elem()
	tmp.Set(vi)
	vi.Set(vj)
	vj.Set(tmp)
}

// Sort sorts an array or slice of structs or pointer to
// structs by comparing the given key, which must be a
// an exported struct field or an exported method with no
// arguments and just one return value. If the key is
// prefixed by the character '-', the sorting is performed
// in descending order.
func Sort(data interface{}, key string) (err error) {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("can't short type %v, must be slice or array", val.Type())
	}
	if val.Len() == 0 {
		return nil
	}
	descending := false
	if key != "" && key[0] == '-' {
		descending = true
		key = key[1:]
	}
	var fn func(reflect.Value) reflect.Value
	// Check the first item to see if we're using
	// a method or a value.
	item := val.Index(0)
	if method := item.MethodByName(key); method.IsValid() {
		fn = methodValue(key)
	} else if field := reflect.Indirect(item).FieldByName(key); field.IsValid() {
		fn = fieldValue(key)
	} else {
		return fmt.Errorf("%T does not have a field nor a method named %q", item.Interface(), key)
	}
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	sort.Sort(&sortable{val, fn, descending})
	return err
}
