package structs

import (
	"reflect"
	"testing"
)

type tagCase struct {
	tag   string
	value *Tag
	err   string
}

var (
	cases = []tagCase{
		{
			",label:Message,help:'Please, enter a message'",
			&Tag{name: "", values: map[string]string{"label": "Message", "help": "Please, enter a message"}},
			"",
		},
		{
			"foo,label:bar,help:let's rock",
			&Tag{name: "foo", values: map[string]string{"label": "bar", "help": "let's rock"}},
			"",
		},
		{
			"foo,label:bar,help:'let\\'s rock'",
			&Tag{name: "foo", values: map[string]string{"label": "bar", "help": "let's rock"}},
			"",
		},
		{
			"-",
			&Tag{name: "-", values: map[string]string{}},
			"",
		},
		{
			",inv'alidkey",
			nil,
			"illegal character ' in key at 4",
		},
		{
			"inv'alidkey",
			nil,
			"illegal character ' in key at 3",
		},
	}
)

func TestParseTags(t *testing.T) {
	for _, v := range cases {
		tag, err := ParseTag(v.tag)
		if err != nil {
			if v.err == "" {
				t.Error(err)
			} else {
				if v.err != err.Error() {
					t.Errorf("expecting error %v, got %v", v.err, err)
				}
			}
		} else {
			if v.err != "" {
				t.Errorf("expecting error %v, got value %v", v.err, tag)
			} else if !reflect.DeepEqual(v.value, tag) {
				t.Errorf("different tags: want %v, got %v", v.value, tag)
			}
		}
	}
}
