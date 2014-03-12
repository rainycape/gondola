package generic

import (
	"reflect"
	"testing"
)

type functionalTest struct {
	in     interface{}
	start  interface{}
	f      interface{}
	expect interface{}
}

var (
	filterTests = []functionalTest{
		{[]int{1, 2, 3, 4}, nil, func(a int) bool { return a >= 4 }, []int{4}},
		{[]int{1, 2, 3, 4}, nil, func(a int) bool { return a > 4 }, []int{}},
		{[]string{"go", "", "go", "go", "go"}, nil, func(a string) bool { return a != "go" }, []string{""}},
	}
	mapTests = []functionalTest{
		{[]int{1, 2, 3, 4}, nil, func(a int) int { return a * 2 }, []int{2, 4, 6, 8}},
		{[]int{1, 2, 3, 4}, nil, func(a int) float64 { return float64(a * 2) }, []float64{2, 4, 6, 8}},
	}
	reduceTests = []functionalTest{
		{[]int{1, 2, 3, 4}, 0, func(a int, b int) int { return a + b }, int(10)},
		{[]int{1, 2, 3, 4}, 1, func(a int, b int) int { return a * b }, int(24)},
	}
)

func TestFilter(t *testing.T) {
	for _, v := range filterTests {
		out := Filter(v.in, v.f)
		if !reflect.DeepEqual(out, v.expect) {
			t.Errorf("expecting %v, got %v instead", v.expect, out)
		}
	}
}

func TestMap(t *testing.T) {
	for _, v := range mapTests {
		out := Map(v.in, v.f)
		if !reflect.DeepEqual(out, v.expect) {
			t.Errorf("expecting %v, got %v instead", v.expect, out)
		}
	}
}

func TestReduce(t *testing.T) {
	for _, v := range reduceTests {
		out := Reduce(v.in, v.start, v.f)
		if !reflect.DeepEqual(out, v.expect) {
			t.Errorf("expecting %v, got %v instead", v.expect, out)
		}
	}
}
