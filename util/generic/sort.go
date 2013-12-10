package generic

import (
	"fmt"
	"reflect"
	"sort"
)

type sortable struct {
	value reflect.Value
	fn    mapFunc
	cmp   lessFunc
}

func (s *sortable) Len() int {
	return s.value.Len()
}

func (s *sortable) less(i, j int) bool {
	fi := s.fn(s.value.Index(i))
	fj := s.fn(s.value.Index(j))
	return s.cmp(fi, fj)
}

func (s *sortable) Less(i, j int) bool {
	return s.less(i, j)
}

func (s *sortable) Swap(i, j int) {
	vi := s.value.Index(i)
	vj := s.value.Index(j)
	tmp := reflect.New(vi.Type()).Elem()
	tmp.Set(vi)
	vi.Set(vj)
	vj.Set(tmp)
}

type reverseSortable struct {
	*sortable
}

func (s *reverseSortable) Less(i, j int) bool {
	return !s.less(i, j)
}

// Sort sorts an array or slice of structs or pointer to
// structs by comparing the given key, which must be a
// an exported struct field or an exported method with no
// arguments and just one return value. If the key is
// prefixed by the character '-', the sorting is performed
// in descending order. If there are any errors, Sort panics
// since they can't be anything but programming errors.
func Sort(data interface{}, key string) {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		panic(fmt.Errorf("can't short type %v, must be slice or array", val.Type()))
	}
	if val.Len() == 0 {
		return
	}
	descending := false
	if key != "" && key[0] == '-' {
		descending = true
		key = key[1:]
	}
	elem := val.Type().Elem()
	fn, typ, err := mapper(key, elem)
	if err != nil {
		panic(err)
	}
	cmp, err := lessComparator(typ)
	if err != nil {
		panic(err)
	}
	if descending {
		sort.Sort(&reverseSortable{&sortable{val, fn, cmp}})
	} else {
		sort.Sort(&sortable{val, fn, cmp})
	}
}
