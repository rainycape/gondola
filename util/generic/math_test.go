package generic

import (
	"reflect"
	"testing"
)

type mathTest struct {
	values interface{}
	min    interface{}
	max    interface{}
	sum    interface{}
	avg    interface{}
}

var (
	mathTests = []mathTest{
		{[]int{}, 0, 0, 0, 0},
		{[]int{0, -7, 10, -20, 44}, int(-20), int(44), int(27), int(5)},
		{[]uint{7, 1, 7, 10, 20, 44}, uint(1), uint(44), uint(89), uint(14)},
		{[]float32{0, -7, 10, -20, 44}, float32(-20), float32(44), float32(27), float32(5.4)},
		{[]float64{0, -7, 10, -20, 44}, float64(-20), float64(44), float64(27), float64(5.4)},
		{[]string{"a", "b", "c"}, "a", "c", "abc", nil},
		// Slices
		{[][]int{[]int{0, 0, 0}, []int{1}, []int{2, 3}}, []int{1}, []int{0, 0, 0}, []int{0, 0, 0, 1, 2, 3}, nil},
	}
)

func TestMin(t *testing.T) {
	for _, v := range mathTests {
		out := Min(v.values)
		if !reflect.DeepEqual(v.min, out) {
			t.Errorf("expecting min(%v) = %v, got %v instead", v.values, v.min, out)
		}
	}
}

func TestMax(t *testing.T) {
	for _, v := range mathTests {
		out := Max(v.values)
		if !reflect.DeepEqual(v.max, out) {
			t.Errorf("expecting max(%v) = %v, got %v instead", v.values, v.max, out)
		}
	}
}

func TestSum(t *testing.T) {
	for _, v := range mathTests {
		out := Sum(v.values)
		if !reflect.DeepEqual(v.sum, out) {
			t.Errorf("expecting sum(%v) = %v, got %v instead", v.values, v.sum, out)
		}
	}
}

func TestAvg(t *testing.T) {
	for _, v := range mathTests {
		if v.avg == nil {
			continue
		}
		out := Avg(v.values)
		if !reflect.DeepEqual(v.avg, out) {
			t.Errorf("expecting avg(%v) = %v, got %v instead", v.values, v.avg, out)
		}
	}
}
