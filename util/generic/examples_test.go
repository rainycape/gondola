package generic

import (
	"fmt"
)

func ExampleSum() {
	in := []int{1, 2, 3, 4}
	out := Sum(in).(int)
	fmt.Println(out)
	// Output: 10
}

func ExampleSum_slice() {
	in := [][]int{[]int{1, 2}, []int{3, 4}}
	out := Sum(in).([]int)
	fmt.Println(out)
	// Output: [1 2 3 4]
}

func ExampleFilter_even() {
	in := []int{1, 2, 3, 4}
	out := Filter(in, func(a int) bool { return a%2 == 0 }).([]int)
	fmt.Println(out)
	// Output: [2 4]
}

func ExampleMap_square() {
	in := []int{1, 2, 3, 4}
	out := Map(in, func(a int) int { return a * a }).([]int)
	fmt.Println(out)
	// Output: [1 4 9 16]
}

func ExampleMap_add_float() {
	in := []int{1, 2, 3, 4}
	out := Map(in, func(a int) float64 { return float64(a) + 0.1 }).([]float64)
	fmt.Println(out)
	// Output: [1.1 2.1 3.1 4.1]
}

func ExampleMap_extract_type() {
	in := []interface{}{1, 2, 3, 4}
	out := Map(in, func(a interface{}) int { return a.(int) }).([]int)
	fmt.Println(out)
	// Output: [1 2 3 4]
}

func ExampleReduce_sum() {
	in := []int{1, 2, 3, 4}
	out := Reduce(in, 0, func(a, b int) int { return a + b }).(int)
	fmt.Println(out)
	// Output: 10
}

func ExampleReduce_mult() {
	in := []float64{-1, 3, 4}
	out := Reduce(in, 1, func(a, b float64) float64 { return a * b }).(float64)
	fmt.Println(out)
	// Output: -12
}

func ExampleReduce_concat() {
	in := []string{"a", "b", "c"}
	out := Reduce(in, "", func(a, b string) string { return a + b }).(string)
	fmt.Println(out)
	// Output: abc
}
