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

type T interface {
	Error(...interface{})
	Errorf(string, ...interface{})
	Fatal(...interface{})
	Logf(string, ...interface{})
}

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

type Data struct {
	Id   int64 `sql:",primary_key,auto_increment"`
	Data []byte
}

type Rectangle struct {
	X      int
	Y      int
	Width  int
	Height int
	// Field must be tagged with name -, since the ORM does not allow
	// implicitely unexported fields.
	ignored int `json:"-"`
}

type Container struct {
	Id    int64       `sql:",primary_key,auto_increment"`
	Rects []Rectangle `sql:",codec:json"`
}

type Array struct {
	Id     int64   `sql:",primary_key,auto_increment"`
	Values []int64 `sql:",codec:gob"`
}

func testOrm(t T, o *Orm, logging bool) {
	if logging {
		// Set logger
		o.SetLogger(log.Std)
		log.SetLevel(log.LDebug)
	} else {
		log.SetLevel(log.LInfo)
	}

	// Register all models first
	TestTable := o.MustRegister((*Test)(nil), &Options{
		Indexes: Indexes(Index("Name", "Value", "Number").Set(DESC, []string{"Name"})),
	})
	o.MustRegister((*Data)(nil), nil)
	o.MustRegister((*Container)(nil), nil)
	o.MustRegister((*Array)(nil), nil)

	// Some basic tests
	now := time.Now()
	o.MustCommitTables()
	obj1 := &Test{
		Name:      "Test1",
		Value:     "Test1",
		Timestamp: now,
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
	res := o.MustSave(obj2)
	id, err := res.LastInsertId()
	if err != nil {
		t.Error(err)
	}
	if id != obj2.Id {
		t.Errorf("Expected id %d, got %d instead", obj2.Id, id)
	}
	count, err := o.Table(TestTable).Count()
	if err != nil {
		t.Error(err)
	}
	if count != 3 {
		t.Errorf("Invalid object count, expected %v, got %v", 3, count)
	}
	var out *Test
	err = o.All().Sort("Id", DESC).One(&out)
	if err != nil {
		t.Error(err)
	} else {
		if out.Id != obj2.Id {
			t.Errorf("Error sorting, expected id %v, got %v", obj2.Id, out.Id)
		}
	}
	err = o.All().Sort("Id", ASC).One(&out)
	if err != nil {
		t.Error(err)
	} else {
		if out.Id != obj1.Id {
			t.Errorf("Error sorting, expected id %v, got %v", obj1.Id, out.Id)
		}
	}
	for _, v := range []int64{2, 3} {
		err := o.One(Eq("Id", v), obj2)
		if err != nil {
			t.Error(err)
		}
		if !obj2.Timestamp.IsZero() {
			t.Errorf("Expected zero timestamp, got %v instead", obj2.Timestamp)
		}
	}
	if _, err := o.DeleteFrom(TestTable, Eq("Id", 2)); err != nil {
		t.Errorf("error deleting with query: %s", err)
	}
	count2, err := o.Table(TestTable).Count()
	if err != nil {
		t.Error(err)
	}
	if count2 != 2 {
		t.Errorf("Invalid object count, expected %v, got %v", 2, count2)
	}
	var obj3 Test
	q := Eq("Id", 1)
	err = o.One(q, &obj3)
	if err != nil {
		t.Error(err)
	}
	if obj3.Id != obj1.Id {
		t.Errorf("invalid ID %v, expected %v.", obj3.Id, obj1.Id)
	}
	// Compare seconds, since some backends (like sqlite) loss subsecond precission
	if !obj3.Timestamp.Truncate(time.Second).Equal(obj1.Timestamp.Truncate(time.Second)) {
		t.Errorf("invalid timestamp %v, expected %v.", obj3.Timestamp, obj1.Timestamp)
	}
	t.Logf("OBJ1 from DB %+v", obj3)
	var obj4 *Test
	err = o.One(q, &obj4)
	if err != nil {
		t.Error(err)
	}
	if obj4.Id != obj1.Id {
		t.Errorf("invalid ID %v, expected %v.", obj4.Id, obj1.Id)
	}
	// Test an object with []byte
	d := []byte("foobar")
	data := &Data{
		Data: d,
	}
	o.MustSave(&data)
	o.MustSave(&Data{})
	err = o.One(Eq("Id", 1), &data)
	if err != nil {
		t.Error(err)
	}
	if string(data.Data) != string(d) {
		t.Errorf("Invalid data %v, expected %v.", data.Data, d)
	}
	err = o.One(Eq("Id", 2), &data)
	if err != nil {
		t.Error(err)
	}
	if data.Data != nil {
		t.Errorf("Invalid data %v, expected %v.", data.Data, nil)
	}
	// Test json-encoded fields
	c := &Container{
		Rects: []Rectangle{
			{1, 2, 3, 4, 5},
		},
	}
	o.MustSave(c)
	err = o.One(Eq("Id", 1), &c)
	if err != nil {
		t.Error(err)
	} else {
		if c.Rects == nil {
			t.Errorf("JSON field not loaded")
		} else {
			r := c.Rects[0]
			if r.X != 1 || r.Y != 2 || r.Width != 3 || r.Height != 4 || r.ignored != 0 {
				t.Errorf("Invalid JSON rect loaded %+v", r)
			}
		}
	}
	arr1 := &Array{Values: []int64{1, 2, 3, 4, 5}}
	o.MustSave(arr1)
	var arr2 *Array
	err = o.One(Eq("Id", 1), &arr2)
	if err != nil {
		t.Error(err)
	} else {
		if !reflect.DeepEqual(arr1.Values, arr2.Values) {
			t.Errorf("Invalid gob decoded field. Want %v, got %v.", arr1.Values, arr2.Values)
		}
	}
	o.Close()
}

func testSqlite(t T, logging bool) {
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
	testOrm(t, o, logging)
}

func TestSqlite(t *testing.T) {
	testSqlite(t, true)
}

func TestPostgresql(t *testing.T) {
	exec.Command("dropdb", "gotest").Run()
	exec.Command("createdb", "gotest").Run()
	o, err := Open("postgres", "dbname=gotest user=fiam password=fiam")
	if err != nil {
		t.Fatal(err)
	}
	testOrm(t, o, true)
}

func BenchmarkSqlite(b *testing.B) {
	for ii := 0; ii < b.N; ii++ {
		// Clear registry
		testSqlite(b, false)
		_nameRegistry = map[string]nameRegistry{}
		_typeRegistry = map[string]typeRegistry{}
	}
}
