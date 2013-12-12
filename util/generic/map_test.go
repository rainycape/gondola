package generic

import (
	"sort"
	"testing"
)

const count = 1000 * 1000

var dict map[int]int

func testMap(t *testing.T, f func(interface{}, interface{})) {
	var out []int
	f(dict, &out)
	if len(out) != len(dict) {
		t.Fatalf("expecting %d elements, got %d instead", len(dict), len(out))
	}
	sort.Ints(out)
	for ii := 0; ii < count; ii++ {
		if out[ii] != ii {
			t.Errorf("bad value at index %d: %d", ii, out[ii])
		}
	}
}

func TestKeys(t *testing.T) {
	testMap(t, Keys)
}

func TestValues(t *testing.T) {
	testMap(t, Values)
}

func BenchmarkKeys(b *testing.B) {
	var out []int
	Keys(dict, &out)
}

func BenchmarkKeysNonGeneric(b *testing.B) {
	out := make([]int, 0, len(dict))
	for k := range dict {
		out = append(out, k)
	}
}

func init() {
	dict = make(map[int]int)
	for ii := 0; ii < count; ii++ {
		dict[ii] = ii
	}
}
