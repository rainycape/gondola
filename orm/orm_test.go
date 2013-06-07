package orm

import (
	"gondola/log"
	_ "gondola/orm/drivers/postgres"
	_ "gondola/orm/drivers/sqlite"
	"io/ioutil"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

type Inner struct {
	Id    int64 `sql:","`
	Value int64 `sql:",nullzero"`
}

type Test struct {
	Inner
	Id        int64  `sql:",primary_key,auto_increment"`
	Name      string `sql:",index,notnull"`
	Value     string `sql:"val"`
	Number    int64  `sql:",nullzero"`
	S         string `sql:",nullzero"`
	Generated string `sql:"-"`
	Timestamp time.Time
}

type PtrTest struct {
	Id        *int64  `sql:",primary_key,auto_increment"`
	Value     *string `sql:"val"`
	Number    *int64  `sql:",nullzero"`
	S         *string `sql:",nullzero"`
	Timestamp *time.Time
}

func testOrm(t *testing.T, o *Orm) {
	// Clear registry
	nameRegistry = map[string]*Model{}
	typeRegistry = map[reflect.Type]*Model{}
	// Set logger
	o.SetLogger(log.Std)

	TestModel := o.MustRegister(&Test{}, nil)
	o.MustCommitModels()
	obj1 := &Test{
		Name:      "Test1",
		Value:     "Test1",
		Timestamp: time.Now(),
	}
	o.MustInsert(obj1)
	if obj1.Id != 1 {
		t.Errorf("invalid ID for object. Expected %v, got %v.", 1, obj1.Id)
	}
	obj2 := &Test{
		Name:  "Test2",
		Value: "Test2",
	}
	o.MustInsert(obj2)
	obj2.Id = 3
	// This should perform an insert, even when it has a primary key
	// because the update will have 0 rows affected.
	o.MustSave(obj2)
	for _, v := range []int64{2, 3} {
		err := o.One(obj2, Eq("Id", v))
		if err != nil {
			t.Error(err)
		}
		if !obj2.Timestamp.IsZero() {
			t.Errorf("Expected zero timestamp, got %v instead", obj2.Timestamp)
		}
	}
	if _, err := o.Delete(TestModel, Eq("Id", 2)); err != nil {
		t.Errorf("error deleting with query: %s", err)
	}
	var obj3 Test
	q := Eq("Id", 1)
	err := o.One(&obj3, q)
	if err != nil {
		t.Error(err)
	}
	if obj3.Id != obj1.Id {
		t.Errorf("invalid ID %v, expected %v.", obj3.Id, obj1.Id)
	}
	t.Logf("OBJ1 from DB %+v", obj3)
	var obj4 *Test
	err = o.One(&obj4, q)
	if err != nil {
		t.Error(err)
	}
	if obj4.Id != obj1.Id {
		t.Errorf("invalid ID %v, expected %v.", obj4.Id, obj1.Id)
	}
	o.Close()
}

func TestSqlite(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	t.Logf("Using db file %s", name)
	f.Close()
	//	defer os.Remove(name)
	o, err := Open("sqlite", name)
	if err != nil {
		t.Fatal(err)
	}
	o.SqlDB().Exec("PRAGMA journal_mode = WAL")
	testOrm(t, o)
}

func TestPostgresql(t *testing.T) {
	exec.Command("dropdb", "gotest").Run()
	exec.Command("createdb", "gotest").Run()
	o, err := Open("postgres", "dbname=gotest user=fiam password=fiam")
	if err != nil {
		t.Fatal(err)
	}
	testOrm(t, o)
}

func init() {
	log.SetLevel(log.LDebug)
}
