package textutil

import (
	"reflect"
	"testing"
)

func TestSplitFields(t *testing.T) {
	cases := map[string][]string{
		"The, quick, brown":             []string{"The", "quick", "brown"},
		"'fo\"x', 'jum,ps', \"ov',er\"": []string{"fo\"x", "jum,ps", "ov',er"},
	}
	for k, v := range cases {
		fields, err := SplitFields(k, ",", "'\"")
		if err != nil {
			t.Errorf("error splitting %q: %s", k, err)
			continue
		}
		if !reflect.DeepEqual(fields, v) {
			t.Errorf("error splitting %q. wanted %v (%d values), got %v instead (%d values)", k, v, len(v), fields, len(fields))
		}
	}
}
