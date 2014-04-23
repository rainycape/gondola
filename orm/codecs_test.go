package orm

import (
	"reflect"
	"testing"
)

type InvalidCodec1 struct {
	// A codec which doesn't exist
	Value []int64 `orm:",codec=nonexistent"`
}

// This type can be correctly encoded by codecs

type Rect struct {
	A, B, C, D int
	ignored    int `orm:"-"`
}

type JsonEncoded struct {
	Id    int64  `orm:",primary_key,auto_increment"`
	Rects []Rect `orm:",codec=json"`
}

type GobEncoded struct {
	Id    int64  `orm:",primary_key,auto_increment"`
	Rects []Rect `orm:",codec=gob"`
}

func testCodecs(t *testing.T, o *Orm) {
	o.mustRegister((*JsonEncoded)(nil), nil)
	o.mustRegister((*GobEncoded)(nil), nil)
	o.mustInitialize()
	rects := []Rect{{A: 1, B: 2, C: 3, D: 4}, {A: 2, B: 3, C: 4, D: 5}}
	j1 := &JsonEncoded{Rects: rects}
	o.MustSave(j1)
	id1 := j1.Id
	var j2 *JsonEncoded
	_, err := o.One(Eq("Id", id1), &j2)
	if err != nil {
		t.Error(err)
	} else if j2 == nil {
		t.Error("j2 is nil")
	} else if !reflect.DeepEqual(j2.Rects, rects) {
		t.Errorf("invalid JSON decoded field. Want %v, got %v.", rects, j2.Rects)
	}
	g1 := &GobEncoded{Rects: rects}
	o.MustSave(g1)
	id2 := g1.Id
	var g2 *GobEncoded
	_, err = o.One(Eq("Id", id2), &g2)
	if err != nil {
		t.Error(err)
	} else if g2 == nil {
		t.Error("g2 is nil")
	} else if !reflect.DeepEqual(g2.Rects, rects) {
		t.Errorf("invalid gob decoded field. Want %v, got %v.", rects, g2.Rects)
	}
}
