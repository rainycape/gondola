package orm

import (
	"gondola/log"
	_ "gondola/orm/drivers/sqlite"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

// This file contains tests that are mostly independent of the ORM drivers.
// In other words, tests in this file are for features in the ORM itself.
// All of them use a temporary SQLite database.

// Interface for testing.B and testing.T
type T interface {
	Error(...interface{})
	Errorf(string, ...interface{})
	Fatal(...interface{})
	Logf(string, ...interface{})
}

type AutoIncrement struct {
	Id int64 `orm:",primary_key,auto_increment"`
	// Must have another field, otherwise there are
	// no fields to insert
	Value string
}

type Timestamp struct {
	Id        int64 `orm:",primary_key,auto_increment"`
	Timestamp time.Time
}

type Object struct {
	Id     int64 `orm:",primary_key,auto_increment"`
	Value  string
	loaded int `orm:"-"`
	saved  int `orm:"-"`
}

func (o *Object) Load() {
	o.loaded++
}

func (o *Object) Save() {
	o.saved++
}

func newOrm(t T, drv, name string, logging bool) *Orm {
	// Clear registry
	_nameRegistry = map[string]nameRegistry{}
	_typeRegistry = map[string]typeRegistry{}
	o, err := Open(drv, name)
	if err != nil {
		t.Fatal(err)
	}
	if logging {
		// Set logger
		o.SetLogger(log.Std)
		log.SetLevel(log.LDebug)
	} else {
		log.SetLevel(log.LInfo)
	}
	return o
}

func newTmpOrm(t T) (string, *Orm) {
	f, err := ioutil.TempFile("", "sqlite-")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	o := newOrm(t, "sqlite", f.Name(), true)
	o.SqlDB().Exec("PRAGMA journal_mode = WAL")
	return f.Name(), o
}

func newMemoryOrm(t T) *Orm {
	o := newOrm(t, "sqlite", ":memory:", true)
	o.SqlDB().Exec("PRAGMA journal_mode = WAL")
	return o
}

func runTest(t *testing.T, f func(*testing.T, *Orm)) {
	name, o := newTmpOrm(t)
	defer o.Close()
	defer os.Remove(name)
	f(t, o)
}

func testAutoIncrement(t *testing.T, o *Orm) {
	o.MustRegister((*AutoIncrement)(nil), nil)
	o.MustCommitTables()
	obj := &AutoIncrement{}
	o.MustSave(obj)
	if obj.Id != 1 {
		t.Errorf("Invalid autoincremented id %v, expected 1", obj.Id)
	}
}

func TestAutoIncrement(t *testing.T) {
	runTest(t, testAutoIncrement)
}

func testTime(t *testing.T, o *Orm) {
	o.MustRegister((*Timestamp)(nil), nil)
	o.MustCommitTables()
	now := time.Now()
	t1 := &Timestamp{}
	o.MustSave(t1)
	t1.Id = 0
	t1.Timestamp = now
	o.MustSave(t1)
	err := o.One(Eq("Id", 1), &t1)
	if err != nil {
		t.Error(err)
	} else {
		if !t1.Timestamp.IsZero() {
			t.Errorf("expected zero timestamp, got %v instead", t1.Timestamp)
		}
	}
	err = o.One(Eq("Id", 2), t1)
	if err != nil {
		t.Error(err)
	} else {
		// Compare seconds, since some backends (like sqlite) loss subsecond precission
		if !t1.Timestamp.Truncate(time.Second).Equal(now.Truncate(time.Second)) {
			t.Errorf("invalid timestamp %v, expected %v.", t1.Timestamp, now)
		}
	}
}

func TestTime(t *testing.T) {
	runTest(t, testTime)
}

func testSaveDelete(t *testing.T, o *Orm) {
	SaveTable := o.MustRegister((*Object)(nil), &Options{
		TableName: "test_save",
	})
	o.MustCommitTables()
	obj := &Object{Value: "Foo"}
	o.MustSaveInto(SaveTable, obj)
	// This should perform an insert, even when it has a primary key
	// because the update will have 0 rows affected.
	obj.Id = 2
	o.MustSaveInto(SaveTable, obj)
	count := o.Table(SaveTable).MustCount()
	if count != 2 {
		t.Errorf("expected count = 2, got %v instead", count)
	}
	// This should perform an update
	obj.Value = "Bar"
	o.MustSaveInto(SaveTable, obj)
	count = o.Table(SaveTable).MustCount()
	if count != 2 {
		t.Errorf("expected count = 2, got %v instead", count)
	}
	var obj2 *Object
	err := o.One(Eq("Id", 2), &obj2)
	if err != nil {
		t.Error(err)
	} else {
		if obj2.Value != obj.Value {
			t.Errorf("bad update, expected value %q, got %q instead", obj.Value, obj2.Value)
		}
	}
	err = o.DeleteFrom(SaveTable, obj)
	if err != nil {
		t.Error(err)
	}
	count = o.Table(SaveTable).MustCount()
	if count != 1 {
		t.Errorf("expected count = 1, got %v instead", count)
	}
	res, err := o.DeleteFromTable(SaveTable, Eq("Id", 1))
	if err != nil {
		t.Error(err)
	} else {
		aff, err := res.RowsAffected()
		if err != nil {
			t.Error(err)
		}
		if aff != 1 {
			t.Errorf("expected 1 affected rows by DELETE, got %v instead", aff)
		}
	}
	count = o.Table(SaveTable).MustCount()
	if count != 0 {
		t.Errorf("expected count = 0, got %v instead", count)
	}
}

func TestSaveDelete(t *testing.T) {
	runTest(t, testSaveDelete)
}
