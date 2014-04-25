package orm

import (
	"testing"
)

func TestAutoIncrement(t *testing.T) {
	runTest(t, testAutoIncrement)
}

func TestBadAutoincrement(t *testing.T) {
	runTest(t, testBadAutoincrement)
}

func TestTime(t *testing.T) {
	runTest(t, testTime)
}

func TestSaveDelete(t *testing.T) {
	runTest(t, testSaveDelete)
}

func TestData(t *testing.T) {
	runTest(t, testData)
}

func TestInnerPointer(t *testing.T) {
	runTest(t, testInnerPointer)
}

func TestTransactions(t *testing.T) {
	runTest(t, testTransactions)
}

func TestFuncTransactions(t *testing.T) {
	runTest(t, testFuncTransactions)
}

func TestQueryAll(t *testing.T) {
	runTest(t, testQueryAll)
}

func TestBadReferences(t *testing.T) {
	runTest(t, testBadReferences)
}

func TestReferences(t *testing.T) {
	runTest(t, testReferences)
}

func TestInvalidCodecs(t *testing.T) {
	runTest(t, testInvalidCodecs)
}

func TestCodecs(t *testing.T) {
	runTest(t, testCodecs)
}

func TestLoadSaveMethods(t *testing.T) {
	runTest(t, testLoadSaveMethods)
}

func TestLoadSaveMethodsErrors(t *testing.T) {
	runTest(t, testLoadSaveMethodsErrors)
}

func TestDefaults(t *testing.T) {
	runTest(t, testDefaults)
}

func BenchmarkLoadSaveMethods(b *testing.B) {
	runBenchmark(b, benchmarkLoadSaveMethods)
}

func benchmarkInsert(b *testing.B, o *Orm) {
	o.mustRegister((*Outer)(nil), &Options{
		Table:   "outer_bench_insert",
		Default: true,
	})
	o.mustInitialize()
	obj := &Outer{
		Key:   "Gondola",
		Inner: &Inner{A: 4, B: 2},
	}
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		obj.Id = 0
		if _, err := o.Insert(obj); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsert(b *testing.B) {
	runBenchmark(b, benchmarkInsert)
}

func benchmarkOne(b *testing.B, o *Orm) {
	o.mustRegister((*Outer)(nil), &Options{
		Table:   "outer_bench_one",
		Default: true,
	})
	o.mustInitialize()
	obj := &Outer{
		Key:   "Gondola",
		Inner: &Inner{A: 4, B: 2},
	}
	if _, err := o.Insert(obj); err != nil {
		b.Fatal(err)
	}
	q := Eq("Id", obj.Id)
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		if _, err := o.One(q, obj); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOne(b *testing.B) {
	runBenchmark(b, benchmarkOne)
}
