package orm

import (
	"gondola/log"
	_ "gondola/orm/drivers/sqlite"
	"testing"
	"time"
)

type Inner struct {
	Id    int64
	Value int64
}

type Test struct {
	Inner
	Id        int64  `sql:",primary_key,auto_increment"`
	Name      string `sql:",index,notnull"`
	Value     string `sql:"val"`
	Generated string `sql:"-"`
	Timestamp time.Time
	private   interface{}
}

func TestOrm(t *testing.T) {
	log.SetLevel(log.LDebug)
	o, err := New("sqlite", "/tmp/foo.db")
	if err != nil {
		t.Fatal(err)
	}
	TestModel := o.MustRegister(&Test{}, nil)
	o.MustCommitModels()
	obj := &Test{
		Name:      "Test1",
		Value:     "Test1",
		Timestamp: time.Now(),
	}
	o.MustInsert(obj)
	log.Infof("TEST 1 %+v", obj)
	obj2 := &Test{
		Name:  "Test2",
		Value: "Test2",
	}
	o.MustInsert(obj2)
	o.Delete(TestModel, Eq("Id", 2))
	log.Infof("TEST 2 %+v", obj2)
	var obj3 Test
	q := Eq("Id", 1)
	err = o.One(&obj3, q)
	if err != nil {
		t.Error(err)
	}
	if obj3.Id != 1 {
		log.Errorf("Invalid id %v", obj3.Id)
	}
	log.Infof("OBJ3 %+v", obj3)
	var obj4 *Test
	err = o.One(&obj4, q)
	if err != nil {
		t.Error(err)
	}
	if obj4.Id != 1 {
		log.Errorf("Invalid id %v", obj4.Id)
	}
	log.Infof("OBJ4 %+v", obj4)
	o.Close()
}
