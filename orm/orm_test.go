package orm

import (
	"bytes"
	"gnd.la/config"
	"gnd.la/log"
	_ "gnd.la/orm/driver/sqlite"
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

type Data struct {
	Id   int64 `orm:",primary_key,auto_increment"`
	Data []byte
}

func (o *Object) Load() {
	o.loaded++
}

func (o *Object) Save() {
	o.saved++
}

type Inner struct {
	A int `orm:",omitempty"`
	B int `orm:",omitempty"`
}

type Outer struct {
	Id    int64 `orm:",primary_key,auto_increment"`
	Key   string
	Inner *Inner
}

type Composite struct {
	Id    int
	Name  string
	Value string
}

func equalTimes(t1, t2 time.Time) bool {
	// Compare seconds, since some backends (like sqlite) loss subsecond precission
	return t1.Truncate(time.Second).Equal(t2.Truncate(time.Second))
}

func newOrm(t T, url string, logging bool) *Orm {
	// Clear registry
	_nameRegistry = map[string]nameRegistry{}
	_typeRegistry = map[string]typeRegistry{}
	o, err := New(config.MustParseURL(url))
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
	o := newOrm(t, "sqlite://"+f.Name(), true)
	o.SqlDB().Exec("PRAGMA journal_mode = WAL")
	return f.Name(), o
}

func newMemoryOrm(t T) *Orm {
	o := newOrm(t, "sqlite://:memory:", true)
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
	o.MustInitialize()
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
	o.MustInitialize()
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
		if !equalTimes(t1.Timestamp, now) {
			t.Errorf("invalid timestamp %v, expected %v.", t1.Timestamp, now)
		}
	}
}

func TestTime(t *testing.T) {
	runTest(t, testTime)
}

func testSaveDelete(t *testing.T, o *Orm) {
	SaveTable := o.MustRegister((*Object)(nil), &Options{
		Table: "test_save",
	})
	o.MustInitialize()
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

func testData(t *testing.T, o *Orm) {
	o.MustRegister((*Data)(nil), &Options{
		Table: "test_data",
	})
	o.MustInitialize()
	data := []byte{1, 2, 3, 4, 5, 6}
	o.MustSave(&Data{Data: data})
	var d *Data
	err := o.One(Eq("Id", 1), &d)
	if err != nil {
		t.Error(err)
	} else {
		if !bytes.Equal(d.Data, data) {
			t.Errorf("invalid stored []byte. Want %v, got %v.", data, d.Data)
		}
	}
}

func TestData(t *testing.T) {
	runTest(t, testData)
}

func testInnerPointer(t *testing.T, o *Orm) {
	o.MustRegister((*Outer)(nil), &Options{
		Table: "test_outer",
	})
	o.MustInitialize()
	out := Outer{Key: "foo"}
	o.MustSave(&out)
	out2 := Outer{Key: "bar", Inner: &Inner{A: 4, B: 2}}
	o.MustSave(&out2)
	var in Outer
	err := o.One(Eq("Key", "foo"), &in)
	if err != nil {
		t.Error(err)
	} else {
		if in.Inner != nil {
			t.Errorf("want %v, got %+v", nil, in.Inner)
		}
	}
	err = o.One(Eq("Key", "bar"), &in)
	if err != nil {
		t.Error(err)
	} else {
		if in.Inner != nil {
			if in.Inner.A != out2.Inner.A {
				t.Errorf("want %v, got %v", out2.Inner.A, in.Inner.A)
			}
			if in.Inner.B != out2.Inner.B {
				t.Errorf("want %v, got %v", out2.Inner.B, in.Inner.A)
			}
		} else {
			t.Errorf("want non-nil, got nil")
		}
	}
}

func TestInnerPointer(t *testing.T) {
	runTest(t, testInnerPointer)
}

func testTransactions(t *testing.T, o *Orm) {
	table := o.MustRegister((*AutoIncrement)(nil), &Options{
		Table: "test_transactions",
	})
	o.MustInitialize()
	obj := &AutoIncrement{}
	tx := o.MustBegin()
	tx.MustSaveInto(table, obj)
	tx.MustCommit()
	e, err := o.Exists(table, Eq("Id", obj.Id))
	if err != nil {
		t.Error(err)
	} else if !e {
		t.Error("commited object does not exist")
	}
	tx2 := o.MustBegin()
	obj.Id = 0
	tx2.MustSaveInto(table, obj)
	tx2.MustRollback()
	e, err = o.Exists(table, Eq("Id", obj.Id))
	if err != nil {
		t.Error(err)
	} else if e {
		t.Error("rolled back object exists")
	}
}

func TestTransactions(t *testing.T) {
	runTest(t, testTransactions)
}

func testCompositePrimaryKey(t *testing.T, o *Orm) {
	// This should fail with a duplicate PK error
	_, err := o.Register((*AutoIncrement)(nil), &Options{
		Table:      "test_composite_fail",
		PrimaryKey: []string{"non-existant"},
	})
	if err == nil {
		t.Error("expecting an error when registering duplicate PK")
	}
	// This should fail because the field can't be mapped
	_, err = o.Register((*Composite)(nil), &Options{
		Table:      "test_composite_fail",
		PrimaryKey: []string{"non-existant"},
	})
	if err == nil {
		t.Error("expecting an error when registering non-existant field as PK")
	}
	table := o.MustRegister((*Composite)(nil), &Options{
		Table:      "test_composite",
		PrimaryKey: []string{"Id", "Name"},
	})
	o.MustInitialize()
	comp := &Composite{
		Id:    1,
		Name:  "Foo",
		Value: "Bar",
	}
	o.MustSave(comp)
	c1, err := o.Count(table, nil)
	if err != nil {
		t.Error(err)
	}
	if c1 != 1 {
		t.Errorf("expecting 1 row, got %v instead", c1)
	}
	_, err = o.InsertInto(table, comp)
	if err == nil {
		t.Error("must return error because of duplicate constraint")
	}
	comp.Value = "Baz"
	o.MustSave(comp)
	var comp2 *Composite
	err = o.Table(table).Filter(Eq("Id", 1)).One(&comp2)
	if err != nil {
		t.Error(err)
	} else if comp2.Value != comp.Value {
		t.Errorf("value not updated. want %q, got %q", comp.Value, comp2.Value)
	}
	comp2.Name = "Go!"
	o.MustSave(comp2)
	c2, err := o.Count(table, nil)
	if err != nil {
		t.Error(err)
	} else if c2 != 2 {
		t.Errorf("expecting 2 rows, got %v instead", c2)
	}
}

func TestCompositePrimaryKey(t *testing.T) {
	runTest(t, testCompositePrimaryKey)
}
