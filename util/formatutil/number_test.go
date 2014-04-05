package formatutil

import (
	"testing"
)

type Languager string

func (l Languager) Language() string {
	return string(l)
}

type numberTest struct {
	in   interface{}
	lang Languager
	out  string
}

var (
	numberTests = []numberTest{
		{1000, "", "1,000"},
		{999, "", "999"},
		{1000.12345, "", "1,000.12345"},
		{999.12345, "", "999.12345"},
		{100000000, "", "100,000,000"},
		{1000, "es", "1.000"},
		{999, "es", "999"},
		{1000.12345, "es", "1.000,12345"},
		{999.12345, "es", "999,12345"},
		{100000000, "es", "100.000.000"},
	}
)

func TestNumber(t *testing.T) {
	for _, v := range numberTests {
		out, err := Number(v.lang, v.in)
		if err != nil {
			t.Error(err)
			continue
		}
		if out != v.out {
			t.Errorf("expecting %q when formatting number %v, got %q instead", v.out, v.in, out)
		}
	}
}
