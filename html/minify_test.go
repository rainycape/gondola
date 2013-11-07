package html

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

type minifyCase struct {
	HTML     string
	Expected string
}

func fromFile(name string) string {
	p := filepath.Join("_test_data", name)
	data, err := ioutil.ReadFile(p)
	if err != nil {
		panic(err)
	}
	return string(data)
}

var (
	minifyCases = []minifyCase{
		{"   <div>   foo     bar \nbaz   </div>", "<div> foo bar baz </div>"},
		{"<div>foo     bar \nbaz </div>", "<div>foo bar baz </div>"},
		{"<div> foo     bar \nbaz</div>", "<div> foo bar baz</div>"},
		{
			"   <div> foo     bar \nbaz </div>  <pre> foo     bar \nbaz </pre>  <div> foo     bar \nbaz </div>",
			"<div> foo bar baz </div><pre> foo     bar \nbaz </pre><div> foo bar baz </div>",
		},
		{
			"<div>You're our 1 millionth visitor <a href=\"http://www.google.com\">click here</a> to claim your price</div>",
			"<div>You're our 1 millionth visitor <a href=\"http://www.google.com\">click here</a> to claim your price</div>",
		},
		{fromFile("nytimes.html"), fromFile("nytimes_mini.html")},
	}
)

func TestMinify(t *testing.T) {
	for _, v := range minifyCases {
		var out bytes.Buffer
		if err := Minify(&out, strings.NewReader(v.HTML)); err != nil {
			t.Errorf("error minifying %q: %s", v.HTML, err)
			continue
		}
		if o := out.String(); o != v.Expected {
			t.Errorf("bad minification: want %q, got %q", v.Expected, o)
		}
	}
}

func BenchmarkMinify(b *testing.B) {
	var out bytes.Buffer
	total := int64(0)
	for _, v := range minifyCases {
		total += int64(len(v.HTML))
	}
	b.SetBytes(total)
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for _, v := range minifyCases {
			if err := Minify(&out, strings.NewReader(v.HTML)); err != nil {
				b.Errorf("error minifying %q: %s", v.HTML, err)
			}
			out.Reset()
		}
	}
}
