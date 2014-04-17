// +build !appengine

package orm

import (
	"testing"
)

func TestAutoIncrement(t *testing.T) {
	runTest(t, testAutoIncrement)
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
	o := newMemoryOrm(t)
	defer o.Close()
	for _, v := range []interface{}{&InvalidCodec1{}} {
		_, err := o.Register(v, nil)
		if err == nil {
			t.Errorf("Expecting an error when registering %T", v)
		}
	}
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

func BenchmarkLoadSaveMethods(b *testing.B) {
	o := newMemoryOrm(b)
	defer o.Close()
	tbl := o.MustRegister((*Object)(nil), &Options{
		Table: "test_load_save_benchmark",
	})
	b.ResetTimer()
	m := tbl.model.fields.Methods
	obj := &Object{}
	for ii := 0; ii < b.N; ii++ {
		m.Load(obj)
	}
}
