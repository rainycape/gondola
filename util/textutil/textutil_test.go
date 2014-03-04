package textutil

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type splitCase struct {
	s      string
	sep    string
	result []string
}

func sepRepr(sep string) string {
	switch sep {
	case "":
		return "SPACE"
	}
	return fmt.Sprintf("%q", sep)
}

func resultRepr(res []string) string {
	values := make([]string, len(res))
	for ii, v := range res {
		values[ii] = fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("[%s] (%d values)", strings.Join(values, ", "), len(values))
}

func TestSplitFields(t *testing.T) {
	cases := []splitCase{
		{"The, quick, brown", ",", []string{"The", "quick", "brown"}},
		{"'fo\"x', 'jum,ps', \"ov',er\"", ",", []string{"fo\"x", "jum,ps", "ov',er"}},
		{"  a\tb\r\nc ", "", []string{"a", "b", "c"}},
		{"''  a\tb\r\nc ", "", []string{"", "a", "b", "c"}},
		{"''  a\tb\r\nc ''   ", "", []string{"", "a", "b", "c", ""}},
	}
	for _, v := range cases {
		fields, err := SplitFields(v.s, v.sep)
		if err != nil {
			t.Errorf("error splitting %q with sep %s: %s", v.s, sepRepr(v.sep), err)
			continue
		}
		if !reflect.DeepEqual(fields, v.result) {
			t.Errorf("error splitting %q with sep %s. wanted %v, got %v", v.s, sepRepr(v.sep), resultRepr(v.result), resultRepr(fields))
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
