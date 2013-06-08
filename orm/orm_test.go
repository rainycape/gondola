package orm

import (
	"gondola/log"
	_ "gondola/orm/drivers/postgres"
	_ "gondola/orm/drivers/sqlite"
	"io/ioutil"
	"os/exec"
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
	Rects []Rectangle `sql:",json"`
}

func testOrm(t *testing.T, o *Orm) {
	// Set logger
	o.SetLogger(log.Std)

	// Register all models first
	TestModel := o.MustRegister((*Test)(nil), &Options{
		Indexes: Indexes(Index("Name", "Value", "Number").Set(DESC, []string{"Name"})),
	})
	_ = o.MustRegister((*Data)(nil), nil)
	_ = o.MustRegister((*Container)(nil), nil)

	// Some basic tests
	now := time.Now()
	o.MustCommitModels()
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
	err = o.One(&obj3, q)
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
	err = o.One(&obj4, q)
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
	err = o.One(&data, Eq("Id", 1))
	if err != nil {
		t.Error(err)
	}
	if string(data.Data) != string(d) {
		t.Errorf("Invalid data %v, expected %v.", data.Data, d)
	}
	err = o.One(&data, Eq("Id", 2))
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
	err = o.One(&c, Eq("Id", 1))
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
