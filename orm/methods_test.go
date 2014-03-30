package orm

import (
	"errors"
	"testing"
)

var (
	loadError = errors.New("no load")
	saveError = errors.New("no save")
)

type LoadError struct {
	Object
}

func (l *LoadError) Load() error {
	return loadError
}

type SaveError struct {
	Object
}

func (s *SaveError) Save() error {
	return saveError
}

func testLoadSaveMethods(t *testing.T, o *Orm) {
	SaveLoadTable := o.MustRegister((*Object)(nil), &Options{
		Table: "test_load_save_methods",
	})
	o.MustInitialize()
	obj := &Object{Value: "Foo"}
	o.MustSaveInto(SaveLoadTable, obj)
	if obj.saved != 1 {
		t.Errorf("Save() was called %d times rather than 1", obj.saved)
	}
	_, err := o.Table(SaveLoadTable).One(&obj)
	if err != nil {
		t.Error(err)
	}
	if obj.loaded != 1 {
		t.Errorf("Load() was called %d times rather than 1", obj.loaded)
	}
	// This performs an update and then an insert, but it
	// should call Save() just once.
	obj.saved = 0
	obj.Id = 2
	o.MustSaveInto(SaveLoadTable, obj)
	if obj.saved != 1 {
		t.Errorf("Save() was called %d times rather than 1", obj.saved)
	}
}

func TestLoadSaveMethods(t *testing.T) {
	runTest(t, testLoadSaveMethods)
}

func testLoadSaveMethodsErrors(t *testing.T, o *Orm) {
	LoadErrorTable := o.MustRegister((*LoadError)(nil), &Options{
		Table: "test_load_error",
	})
	SaveErrorTable := o.MustRegister((*SaveError)(nil), &Options{
		Table: "test_save_error",
	})
	o.MustInitialize()
	_, err := o.SaveInto(SaveErrorTable, &SaveError{})
	if err != saveError {
		t.Errorf("unexpected error %v when saving SaveError", err)
	}
	le := &LoadError{}
	o.MustSaveInto(LoadErrorTable, le)
	_, err = o.Table(LoadErrorTable).Filter(Eq("Object.Id", 1)).One(&le)
	if err != loadError {
		t.Errorf("unexpected error %v when loading LoadError", err)
	}
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
