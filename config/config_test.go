package config

import (
	"reflect"
	"strings"
	"testing"
)

type TDefaultConfig struct {
	A int `default:"7"`
}

type TConfig struct {
	A  int
	B  uint
	C  float64
	D  float32
	E  string
	SA []int
	SB []uint
	SC []float64
	SD []float32
	SE []string
	MA map[int]int
	MB map[uint]uint
	MC map[float64]float64
	MD map[float32]float32
	ME map[string]string
}

type testCase struct {
	input  string
	expect TConfig
}

var (
	cases = []testCase{
		{
			"",
			TConfig{},
		},
		{
			"a = 5",
			TConfig{A: 5},
		},
		{
			"a = 5\nsa = 1,2,3",
			TConfig{A: 5, SA: []int{1, 2, 3}},
		},
		{
			"b = 5\nsb = 1,2,3",
			TConfig{B: 5, SB: []uint{1, 2, 3}},
		},
		{
			"c = 5.3\nsc = 1.1,2.2,3.3",
			TConfig{C: 5.3, SC: []float64{1.1, 2.2, 3.3}},
		},
		{
			"d = 5.3\nsd = 1.1,2.2,3.3",
			TConfig{D: 5.3, SD: []float32{1.1, 2.2, 3.3}},
		},
		{
			"e = go\nse = 1,2,3",
			TConfig{E: "go", SE: []string{"1", "2", "3"}},
		},
		{
			"e = go\nse = \"1,2\",3",
			TConfig{E: "go", SE: []string{"1,2", "3"}},
		},
		{
			"e = go\nse = '1,2',3",
			TConfig{E: "go", SE: []string{"1,2", "3"}},
		},
		{
			"ma = 1=2,3=4\nmb = 1=2,3=4\nmc=3.14=.15\nmd=92=65.0,35=89\nme=a=b,'c,d'=e,f='g'",
			TConfig{
				MA: map[int]int{1: 2, 3: 4},
				MB: map[uint]uint{1: 2, 3: 4},
				MC: map[float64]float64{3.14: 0.15},
				MD: map[float32]float32{92: 65, 35: 89},
				ME: map[string]string{"a": "b", "c,d": "e", "f": "g"},
			},
		},
	}
)

func TestDefaultConfig(t *testing.T) {
	var out TDefaultConfig
	err := ParseReader(strings.NewReader(""), &out)
	if err != nil {
		t.Errorf("error parsing empty :%s", err)
	} else if out.A != 7 {
		t.Errorf("expecting default value 7, got %v instead", out.A)
	}
}

func TestConfig(t *testing.T) {
	for _, v := range cases {
		var out TConfig
		err := ParseReader(strings.NewReader(v.input), &out)
		if err != nil {
			t.Errorf("error parsing config %q: %s", v.input, err)
			continue
		}
		if !reflect.DeepEqual(out, v.expect) {
			t.Errorf("expecting config %v from %q, got %v instead", v.expect, v.input, out)
		}
	}
}
