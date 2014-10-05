package generic

import (
	"reflect"
	"testing"
)

const selectableCount = 1000

type Selectable struct {
	Name string
}

func (s *Selectable) GetName() string {
	return s.Name
}

var (
	selectables []*Selectable
)

func TestSelectField(t *testing.T) {
	names := Select(selectables, "Name").([]string)
	for ii, v := range names {
		if v != selectableName(ii) {
			t.Errorf("bad selection at index %d. want %q, got %q", ii, selectableName(ii), v)
		}
	}
}

func TestSelectMethod(t *testing.T) {
	names := Select(selectables, "GetName").([]string)
	for ii, v := range names {
		if v != selectableName(ii) {
			t.Errorf("bad selection at index %d. want %q, got %q", ii, selectableName(ii), v)
		}
	}
}

func TestContains(t *testing.T) {
	numbers := []int{1, 2, 3, 4, 5}
	for _, v := range numbers {
		if !Contains(numbers, v) {
			t.Errorf("%v should contain %v", numbers, v)
		}
	}
	no := numbers[len(numbers)-1] + 1000
	if Contains(numbers, no) {
		t.Errorf("%v should not contain %v", numbers, no)
	}
}

func TestRemove(t *testing.T) {
	numbers := []int{1, 1, 2, 3, 4, 5}
	if Remove(&numbers, int(7)) {
		t.Fatalf("7 is not in %v", numbers)
	}
	if !Remove(&numbers, int(1)) {
		t.Fatal("should have removed 1")
	}
	if len(numbers) != 5 {
		t.Fatalf("len(numbers) should be 5, not %d", len(numbers))
	}
	if !Remove(&numbers, int(1)) {
		t.Fatal("should have removed 1")
	}
	if len(numbers) != 4 {
		t.Fatalf("len(numbers) should be 4, not %d", len(numbers))
	}
	if Remove(&numbers, int(1)) {
		t.Fatalf("1 is not in %v", numbers)
	}
	Remove(&numbers, int(3))
	Remove(&numbers, int(5))
	expected := []int{2, 4}
	if !reflect.DeepEqual(numbers, expected) {
		t.Fatalf("final result should be %v, not %v", expected, numbers)
	}
}

func BenchmarkSelect(b *testing.B) {
	b.ReportAllocs()
	for ii := 0; ii < b.N; ii++ {
		_ = Select(selectables, "Name").([]string)
	}
}

func BenchmarkSelectNonGeneric(b *testing.B) {
	b.ReportAllocs()
	names := make([]string, len(selectables))
	for ii := 0; ii < b.N; ii++ {
		for jj, n := range selectables {
			names[jj] = n.Name
		}
	}
}

func BenchmarkSelectMethod(b *testing.B) {
	b.ReportAllocs()
	for ii := 0; ii < b.N; ii++ {
		_ = Select(selectables, "GetName").([]string)
	}
}

func BenchmarkSelectMethodNonGeneric(b *testing.B) {
	b.ReportAllocs()
	names := make([]string, len(selectables))
	for ii := 0; ii < b.N; ii++ {
		for jj, n := range selectables {
			names[jj] = n.GetName()
		}
	}
}

func selectableName(idx int) string {
	return string(rune(int('A') + idx))
}

func init() {
	selectables = make([]*Selectable, selectableCount)
	for ii := range selectables {
		selectables[ii] = &Selectable{selectableName(ii)}
	}
}
