package orm

import (
	_ "gondola/orm/drivers/sqlite"
	"reflect"
	"testing"
)

// These types should return an error when trying to register
// them because they use field codecs in an invalid way.

type HasUnexported struct {
	A int64
	b int64
}

type InvalidCodec1 struct {
	// A codec which doesn't exist
	Value []int64 `orm:",codec:nonexistent"`
}

// The following two types try to use a codec with
// structs with unexported fields, which is not allowed.

type InvalidCodec2 struct {
	Value []HasUnexported `orm:",codec:json"`
}

type InvalidCodec3 struct {
	Value []HasUnexported `orm:",codec:gob"`
}

// This type can be correctly encoded by codecs

type Rect struct {
	A, B, C, D int
	ignored    int `orm:"-"`
}

type JsonEncoded struct {
	Id    int64  `orm:",primary_key,auto_increment"`
	Rects []Rect `orm:",codec:json"`
}

type GobEncoded struct {
	Id    int64  `orm:",primary_key,auto_increment"`
	Rects []Rect `orm:",codec:gob"`
}

func TestInvalidCodecs(t *testing.T) {
	o := newMemoryOrm(t)
	defer o.Close()
	for _, v := range []interface{}{&InvalidCodec1{}, &InvalidCodec2{}, &InvalidCodec3{}} {
		_, err := o.Register(v, nil)
		if err == nil {
			t.Errorf("Expecting an error when registering %T", v)
		}
	}
}

func testCodecs(t *testing.T, o *Orm) {
	o.MustRegister((*JsonEncoded)(nil), nil)
	o.MustRegister((*GobEncoded)(nil), nil)
	o.MustInitialize()
	q := Eq("Id", 1)
	rects := []Rect{{A: 1, B: 2, C: 3, D: 4}, {A: 2, B: 3, C: 4, D: 5}}
	j1 := &JsonEncoded{Rects: rects}
	o.MustSave(j1)
	var j2 *JsonEncoded
	err := o.One(q, &j2)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(j2.Rects, rects) {
		t.Errorf("invalid JSON decoded field. Want %v, got %v.", rects, j2.Rects)
	}
	g1 := &GobEncoded{Rects: rects}
	o.MustSave(g1)
	var g2 *GobEncoded
	err = o.One(q, &g2)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(g2.Rects, rects) {
		t.Errorf("invalid gob decoded field. Want %v, got %v.", rects, g2.Rects)
	}
}

func TestCodecs(t *testing.T) {
	runTest(t, testCodecs)
}
