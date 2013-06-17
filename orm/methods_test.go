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
		TableName: "test_load_save_methods",
	})
	o.MustCommitTables()
	obj := &Object{Value: "Foo"}
	o.MustSaveInto(SaveLoadTable, obj)
	if obj.saved != 1 {
		t.Errorf("Save() was called %d times rather than 1", obj.saved)
	}
	err := o.Table(SaveLoadTable).One(&obj)
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
		TableName: "test_load_error",
	})
	SaveErrorTable := o.MustRegister((*SaveError)(nil), &Options{
		TableName: "test_save_error",
	})
	o.MustCommitTables()
	_, err := o.SaveInto(SaveErrorTable, &SaveError{})
	if err != saveError {
		t.Errorf("unexpected error %v when saving SaveError", err)
	}
	le := &LoadError{}
	o.MustSaveInto(LoadErrorTable, le)
	err = o.Table(LoadErrorTable).Filter(Eq("Object.Id", 1)).One(&le)
	if err != loadError {
		t.Errorf("unexpected error %v when loading LoadError", err)
	}
}

func TestLoadSaveMethodsErrors(t *testing.T) {
	runTest(t, testLoadSaveMethodsErrors)
}
