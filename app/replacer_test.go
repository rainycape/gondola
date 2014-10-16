package app

import (
	"strconv"
	"testing"
)

type fakeProvider struct {
}

func (f *fakeProvider) Count() int {
	// Over 9000
	return 9001
}

func (f *fakeProvider) Arg(ii int) string {
	return strconv.Itoa(ii)
}

func (f *fakeProvider) Param(name string) string {
	return name
}

func TestReplacer(t *testing.T) {
	p := &fakeProvider{}
	cases := map[string]string{
		"${0} test":     "0 test",
		"$${0} ${test}": "$${0} test",
		"a ${b} c":      "a b c",
		"a b ${c}":      "a b c",
		"${a}${b}${c}":  "abc",
		"${a}${b}$c":    "ab$c",
	}
	for k, v := range cases {
		t.Logf("pattern %q", k)
		repl := newReplacer(k)
		if repl == nil {
			t.Errorf("pattern %q did not generate a replacer", k)
			continue
		}
		res := repl.Replace(p)
		if res != v {
			t.Errorf("expecting repl %q from pattern %q, got %q instead", v, k, res)
		}
	}
}
