package types

import (
	"errors"
	"reflect"
	"sort"
	"testing"
)

type stringer string

func (s stringer) String() string {
	return string(s)
}

type typesTest struct {
	in  interface{}
	out interface{}
}

var (
	typesTests = []typesTest{
		{"1", 1},
		{1.1, 1},
		{1.99999, 1},
		{true, 1},
		{false, 0},
		{0, 0},
		{uint(42), 42},
		{"300", uint(300)},
		{1.99999, uint(1)},
		{true, uint(1)},
		{false, uint(0)},
		{42, uint(42)},
		{1, 1.0},
		{uint(3), 3.0},
		{uint(3.0), uint(3.0)},
		{"1.1", 1.1},
		{5.0, 5.0},
		{true, 1.0},
		{false, 0.0},
		{nil, ""},
		{true, "true"},
		{1, "1"},
		{1.1, "1.1"},
		{(**int)(nil), ""},
		{"", ""},
		{"go", "go"},
		{errors.New("go"), "go"},
		{stringer("go"), "go"},
	}
	izero      int
	truthTests = []typesTest{
		{0, false},
		{10, true},
		{false, false},
		{true, true},
		{[]int(nil), false},
		{make([]int, 0), false},
		{make([]int, 1), true},
		{"", false},
		{"false", true},
		{typesTest{}, true},
		{(*int)(nil), false},
		{&izero, true},
		{3.14, true},
		{uint(0), false},
		{0i, false},
		{1i, true},
	}
	isIntTests = []typesTest{
		{int(42), true},
		{int8(42), true},
		{int16(42), true},
		{int32(42), true},
		{int64(42), true},
		{uint(42), false},
		{uint8(42), false},
		{uint16(42), false},
		{uint32(42), false},
		{uint64(42), false},
		{float32(42), false},
		{float64(42), false},
		{complex64(42), false},
		{complex128(42), false},
		{"", false},
		{[]int(nil), false},
	}
	isUintTests = []typesTest{
		{int(42), false},
		{int8(42), false},
		{int16(42), false},
		{int32(42), false},
		{int64(42), false},
		{uint(42), true},
		{uint8(42), true},
		{uint16(42), true},
		{uint32(42), true},
		{uint64(42), true},
		{float32(42), false},
		{float64(42), false},
		{complex64(42), false},
		{complex128(42), false},
		{"", false},
		{[]int(nil), false},
	}
	isFloatTests = []typesTest{
		{int(42), false},
		{int8(42), false},
		{int16(42), false},
		{int32(42), false},
		{int64(42), false},
		{uint(42), false},
		{uint8(42), false},
		{uint16(42), false},
		{uint32(42), false},
		{uint64(42), false},
		{float32(42), true},
		{float64(42), true},
		{complex64(42), false},
		{complex128(42), false},
		{"", false},
		{[]int(nil), false},
	}
	isNumericTests = []typesTest{
		{int(42), true},
		{int8(42), true},
		{int16(42), true},
		{int32(42), true},
		{int64(42), true},
		{uint(42), true},
		{uint8(42), true},
		{uint16(42), true},
		{uint32(42), true},
		{float32(42), true},
		{float64(42), true},
		{complex64(42), true},
		{complex128(42), true},
		{"", false},
		{[]int(nil), false},
	}
)

func TestTo(t *testing.T) {
	for _, v := range typesTests {
		switch x := v.out.(type) {
		case int:
			res, err := ToInt(v.in)
			if err != nil {
				t.Error(err)
			}
			if res != x {
				t.Errorf("expecting %v from %v (%T), got %v instead", x, v.in, v.in, res)
			}
		case uint:
			res, err := ToUint(v.in)
			if err != nil {
				t.Error(err)
			}
			if res != x {
				t.Errorf("expecting %v from %v (%T), got %v instead", x, v.in, v.in, res)
			}
		case float64:
			res, err := ToFloat(v.in)
			if err != nil {
				t.Error(err)
			}
			if res != x {
				t.Errorf("expecting %v from %v (%T), got %v instead", x, v.in, v.in, res)
			}
		case string:
			res := ToString(v.in)
			if res != x {
				t.Errorf("expecting %v from %v (%T), got %v instead", x, v.in, v.in, res)
			}
		default:
			t.Errorf("unexpected out type %T", v.out)
		}
	}
	if _, err := ToInt(nil); err == nil {
		t.Errorf("expecting error in ToInt()")
	}
	if _, err := ToUint(nil); err == nil {
		t.Errorf("expecting error in ToUint()")
	}
	if _, err := ToFloat(nil); err == nil {
		t.Errorf("expecting error in ToFloat()")
	}
	s := typesTest{}
	if _, err := ToInt(s); err == nil {
		t.Errorf("expecting error in ToInt()")
	}
	if _, err := ToUint(s); err == nil {
		t.Errorf("expecting error in ToUint()")
	}
	if _, err := ToFloat(s); err == nil {
		t.Errorf("expecting error in ToFloat()")
	}
}

func TestTrue(t *testing.T) {
	for _, v := range truthTests {
		val, ok := IsTrue(v.in)
		if !ok {
			t.Errorf("can't determine truth value for %T", v.in)
			continue
		}
		if val != v.out.(bool) {
			t.Errorf("expecting %v value from %v (%T), got %v", v.out, v.in, v.in, val)
		}
	}
	ret, ok := IsTrue(nil)
	if ret || !ok {
		t.Errorf("invalid value should return false, true")
	}
}

func testTypeTester(t *testing.T, tests []typesTest, f func(reflect.Type) bool) {
	for _, v := range tests {
		val := reflect.ValueOf(v.in)
		exp := v.out.(bool)
		res := f(val.Type())
		if res != exp {
			t.Errorf("expecting %v for %s, got %v", exp, val.Type(), res)
		}
	}
}

func TestIsInt(t *testing.T)     { testTypeTester(t, isIntTests, IsInt) }
func TestIsUint(t *testing.T)    { testTypeTester(t, isUintTests, IsUint) }
func TestIsFloat(t *testing.T)   { testTypeTester(t, isFloatTests, IsFloat) }
func TestIsNumeric(t *testing.T) { testTypeTester(t, isNumericTests, IsNumeric) }

func reflectSortCompare(t *testing.T, sorted interface{}, v reflect.Value) {
	vlist := make([]reflect.Value, v.Len())
	for ii := 0; ii < v.Len(); ii++ {
		vlist[ii] = v.Index(ii)
	}
	if err := SortValues(vlist); err != nil {
		t.Error(err)
		return
	}
	s := reflect.ValueOf(sorted)
	slist := make([]reflect.Value, s.Len())
	for ii := 0; ii < s.Len(); ii++ {
		slist[ii] = s.Index(ii)
	}
	for ii := 0; ii < len(vlist); ii++ {
		exp := slist[ii].Interface()
		got := vlist[ii].Interface()
		if !Equal(exp, got) {
			t.Errorf("expecting %v at index %d, got %v", exp, ii, got)
		}
	}
}

type uintslice []uint

func (u uintslice) Len() int           { return len(u) }
func (u uintslice) Less(i, j int) bool { return u[i] < u[j] }
func (u uintslice) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

func TestSort(t *testing.T) {
	tests := []interface{}{
		[]int{3, 4, 6, 7, 2},
		[]uint{3, 4, 6, 7, 2},
		[]float64{3, 4, 6, 7, 2},
		[]string{"f", "d", "a", "c"},
		[]int{1},
		[]int{},
	}
	for _, v := range tests {
		vv := reflect.ValueOf(v)
		cpy := reflect.MakeSlice(vv.Type(), vv.Len(), vv.Len())
		reflect.Copy(cpy, vv)
		switch x := v.(type) {
		case []int:
			sort.Ints(x)
		case []uint:
			sort.Sort(uintslice(x))
		case []float64:
			sort.Float64s(x)
		case []string:
			sort.Strings(x)
		}
		reflectSortCompare(t, v, cpy)
	}
}

func TestSettable(t *testing.T) {
	var a int
	var b *int
	var c int
	tests := []typesTest{
		{&a, nil},
		{&b, nil},
		{c, ErrCantSet},
		{nil, ErrInvalidValue},
	}
	for _, v := range tests {
		set, err := SettableValue(v.in)
		if err != v.out {
			t.Errorf("expecting error %v, got %v", v.out, err)
			continue
		}
		if v.out == nil && !set.CanSet() {
			t.Errorf("can't set value")
		}
	}
}

func TestConvert(t *testing.T) {
	tests := []typesTest{
		{42.0, 42},
		{42.0, uint(42)},
		{42, 42.0},
		{3.14, "3.14"},
		{[]byte("go"), "go"},
	}
	for _, v := range tests {
		out := reflect.New(reflect.TypeOf(v.out))
		if err := Convert(v.in, out.Interface()); err != nil {
			t.Error(err)
			continue
		}
		if !Equal(out.Elem().Interface(), v.out) {
			t.Errorf("expecting %v, got %v", v.out, out.Elem().Interface())
		}
	}
}
