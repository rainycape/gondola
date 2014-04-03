package stringutil

import (
	"testing"
)

func TestCamelCaseToLower(t *testing.T) {
	cases := map[string]string{
		"FooBarBaz":  "foo_bar_baz",
		"FOOBarBaz":  "foo_bar_baz",
		"TEST":       "test",
		"goLANG":     "go_lang",
		"myINTValue": "my_int_value",
		"":           "",
		"T":          "t",
		"t":          "t",
		"Id":         "id",
		"FóoBar":     "fóo_bar",
	}
	for k, v := range cases {
		if u := CamelCaseToLower(k, "_"); u != v {
			t.Errorf("Error transforming camel case %q to lower. Want %q, got %q.", k, v, u)
		}
	}
}
