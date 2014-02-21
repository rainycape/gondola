package textutil

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitFields(t *testing.T) {
	cases := map[string][]string{
		"The, quick, brown":             []string{"The", "quick", "brown"},
		"'fo\"x', 'jum,ps', \"ov',er\"": []string{"fo\"x", "jum,ps", "ov',er"},
	}
	for k, v := range cases {
		fields, err := SplitFields(k, ",")
		if err != nil {
			t.Errorf("error splitting %q: %s", k, err)
			continue
		}
		if !reflect.DeepEqual(fields, v) {
			t.Errorf("error splitting %q. wanted %v (%d values), got %v instead (%d values)", k, v, len(v), fields, len(fields))
		}
	}
}

type iniTest struct {
	text   string
	expect map[string]string
	err    string
}

func TestIni(t *testing.T) {
	iniTests := []*iniTest{
		{"a = b  \n 3 = 7", map[string]string{"a": "b", "3": "7"}, ""},
		{"a = b  \r\n 3 = 7", map[string]string{"a": "b", "3": "7"}, ""},
		{"a = multiline\\\n value  \n 3 = 7", map[string]string{"a": "multiline value", "3": "7"}, ""},
		{"3 = 7\ninvalid", map[string]string{"a": "multiline value", "3": "7"}, "invalid line 2 \"invalid\" - missing separator \"=\""},
	}
	for _, v := range iniTests {
		res, err := ParseIni(strings.NewReader(v.text))
		if err != nil {
			if v.err != err.Error() {
				if v.err == "" {
					t.Errorf("unexpected error parsing %q: %s", v.text, err)
				} else {
					t.Errorf("expecting error %s parsing %q, got %s instead", v.err, v.text, err)
				}
			}
		} else {
			if v.err != "" {
				t.Errorf("expecting error %s parsing %q, got no error instead", v.err, v.text)
			} else {
				if !reflect.DeepEqual(v.expect, res) {
					t.Errorf("expecting %v parsing %q, got %v instead", v.expect, v.text, res)
				}
			}
		}
	}
}
