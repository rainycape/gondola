package sortutil

import (
	"testing"
)

type Test1 struct {
	Name  string
	Value int
}

var (
	chars = []byte{'A', 'B', 'C', 'D', 'E'}
	tests = []*Test1{
		{"B", 4},
		{"C", 2},
		{"E", 3},
		{"D", 0},
		{"A", 1},
	}
)

func TestField(t *testing.T) {
	var t1 []*Test1
	t1 = append(t1, tests...)
	if err := Sort(t1, "Name"); err != nil {
		t.Fatal(err)
	}
	for ii, v := range t1 {
		if ex := string(chars[ii]); ex != v.Name {
			t.Errorf("bad value at index %d. want %s, got %s", ii, ex, v.Name)
		}
	}
	if err := Sort(t1, "-Name"); err != nil {
		t.Fatal(err)
	}
	for ii, v := range t1 {
		if ex := string(chars[len(chars)-ii-1]); ex != v.Name {
			t.Errorf("bad value at index %d. want %s, got %s", ii, ex, v.Name)
		}
	}
}
