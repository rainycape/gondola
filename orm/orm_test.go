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
	Name      string `sql:",unique,index,notnull"`
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
	o.MustRegister(&Test{}, nil)
	o.MustCommitModels()
	obj := &Test{
		Name:      "Test1",
		Value:     "Test1",
		Timestamp: time.Now(),
	}
	o.MustInsert(obj)
	log.Infoln("TEST 1 %+v", obj)
	obj2 := &Test{
		Name:  "Test2",
		Value: "Test2",
	}
	o.MustInsert(obj2)
	log.Infoln("TEST 2 %+v", obj2)
	o.Close()
}
