package generic

import (
	"sort"
	"testing"
)

type Test1 struct {
	Name  string
	Value int
}

func (t *Test1) GetName() string {
	return t.Name
}

func (t *Test1) GetValue() int {
	return t.Value
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
	long []*Test1
)

func TestField(t *testing.T) {
	var t1 []*Test1
	t1 = append(t1, tests...)
	Sort(t1, "Name")
	for ii, v := range t1 {
		if ex := string(chars[ii]); ex != v.Name {
			t.Errorf("bad value at index %d. want %s, got %s", ii, ex, v.Name)
		}
	}
	Sort(t1, "-Name")
	for ii, v := range t1 {
		if ex := string(chars[len(chars)-ii-1]); ex != v.Name {
			t.Errorf("bad value at index %d. want %s, got %s", ii, ex, v.Name)
		}
	}
}

func TestMethod(t *testing.T) {
	var t1 []*Test1
	t1 = append(t1, tests...)
	Sort(t1, "GetName")
	for ii, v := range t1 {
		if ex := string(chars[ii]); ex != v.Name {
			t.Errorf("bad value at index %d. want %s, got %s", ii, ex, v.Name)
		}
	}
	Sort(t1, "-GetName")
	for ii, v := range t1 {
		if ex := string(chars[len(chars)-ii-1]); ex != v.Name {
			t.Errorf("bad value at index %d. want %s, got %s", ii, ex, v.Name)
		}
	}
}

func BenchmarkLong(b *testing.B) {
	b.ReportAllocs()
	for ii := 0; ii < b.N; ii++ {
		b.StopTimer()
		var s []*Test1
		s = append(s, long...)
		b.StartTimer()
		Sort(s, "Value")
	}
}

type test1Slice []*Test1

func (t test1Slice) Len() int {
	return len(t)
}

func (t test1Slice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t test1Slice) Less(i, j int) bool {
	return t[i].Value < t[j].Value
}

func BenchmarkLongNonReflect(b *testing.B) {
	b.ReportAllocs()
	for ii := 0; ii < b.N; ii++ {
		b.StopTimer()
		var s []*Test1
		s = append(s, long...)
		b.StartTimer()
		sort.Sort(test1Slice(s))
	}
}

func init() {
	long = make([]*Test1, 1000)
	for ii := range long {
		long[ii] = &Test1{Value: len(long) - ii}
	}
}
