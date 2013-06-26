package types

import (
	"reflect"
	"testing"
)

type ParseCase struct {
	Value    string
	Type     reflect.Type
	Expected interface{}
}

func TestParse(t *testing.T) {
	cases := []ParseCase{
		{"1", reflect.TypeOf(int(0)), 1},
		{"1", reflect.TypeOf(true), true},
		{"0", reflect.TypeOf(true), false},
		{"false", reflect.TypeOf(true), false},
		{"", reflect.TypeOf(true), false},
		{"", reflect.TypeOf(int(0)), 0},
		{"2000", reflect.TypeOf(uint8(0)), uint8(255)},
		{"-2000", reflect.TypeOf(int8(0)), int8(-128)},
		{"2000", reflect.TypeOf(int8(0)), int8(127)},
		{"56.950000", reflect.TypeOf(float64(0)), 56.95},
		{"foo", reflect.TypeOf("bar"), "foo"},
	}
	for _, v := range cases {
		val := reflect.New(v.Type)
		err := Parse(v.Value, val.Interface())
		if err != nil {
			t.Errorf("Error parsing %q: %s", v.Value, err)
			continue
		}
		result := val.Elem().Interface()
		if !reflect.DeepEqual(result, v.Expected) {
			t.Errorf("Error parsing %q. Want %v, got %v.", v.Value, v.Expected, result)
			continue
		}
		t.Logf("Parsed %q as %v", v.Value, result)
	}
}
