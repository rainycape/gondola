package generic

import (
	"fmt"
	"reflect"
	"sort"
)

type sortable struct {
	length int
	value  handle
	fn     mapFunc
	cmp    lessFunc
	idx    indexFunc
	sw     swapFunc
}

func (s *sortable) Len() int {
	return s.length
}

func (s *sortable) Less(i, j int) bool {
	vi := s.fn(s.idx(s.value, i))
	vj := s.fn(s.idx(s.value, j))
	return s.cmp(vi, vj)
}

func (s *sortable) Swap(i, j int) {
	s.sw(s.value, i, j)
}

type reverseSortable struct {
	*sortable
}

func (s *reverseSortable) Less(i, j int) bool {
	return !s.sortable.Less(i, j)
}

// Sort sorts an array or slice of structs or pointer to
// structs by comparing the given key, which must be a
// an exported struct field or an exported method with no
// arguments and just one return value. If the key is
// prefixed by the character '-', the sorting is performed
// in descending order. If there are any errors, Sort panics
// since they can't be anything but programming errors.
func Sort(data interface{}, key string) {
	descending := false
	if key != "" && key[0] == '-' {
		descending = true
		key = key[1:]
	}
	fn, val, elem, typ, err := sliceMapper(data, key)
	if err != nil {
		panic(err)
	}
	if fn == nil {
		// Empty slice
		return
	}
	cmp, err := lessComparator(typ)
	if err != nil {
		panic(err)
	}
	srt := &sortable{val.Len(), getHandle(val), fn, cmp, indexer(elem), swapper(elem)}
	if descending {
		sort.Sort(&reverseSortable{srt})
	} else {
		sort.Sort(srt)
	}
}

type funcSortable struct {
	val reflect.Value
	fn  reflect.Value
}

func (fs *funcSortable) Len() int {
	return fs.val.Len()
}

func (fs *funcSortable) Less(i, j int) bool {
	vi := fs.val.Index(i)
	vj := fs.val.Index(j)
	res := fs.fn.Call([]reflect.Value{vi, vj})
	return res[0].Bool()
}

func (fs *funcSortable) Swap(i, j int) {
	vi := fs.val.Index(i)
	vj := fs.val.Index(j)
	tmp := reflect.New(vi.Type()).Elem()
	tmp.Set(vi)
	vi.Set(vj)
	vj.Set(tmp)
}

var boolType = reflect.TypeOf(true)

// SortFunc shorts the given slice or array using the provided less
// function. The function must accept two arguments of the same type
// of the slice element and must return just one bool argument.
func SortFunc(data interface{}, less interface{}) {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Array && val.Kind() != reflect.Slice {
		panic(fmt.Errorf("first argument to SortFunc must be slice or array, not %T", data))
	}
	fn := reflect.ValueOf(less)
	if fn.Kind() != reflect.Func {
		panic(fmt.Errorf("second argument to SortFunc must be func, not %T", less))
	}
	elem := val.Type().Elem()
	ft := fn.Type()
	if ft.NumIn() != 2 || ft.In(0) != elem || ft.In(1) != elem || ft.NumOut() != 1 || ft.Out(0) != boolType {
		panic(fmt.Errorf("less function for %s must be func(%s, %s) bool, not %s", val.Type(), elem, elem, ft))
	}
	sort.Sort(&funcSortable{val, fn})
}
