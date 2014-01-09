package template

import (
	"bytes"
	"gnd.la/loaders"
	"testing"
)

type functionTest struct {
	tmpl   string
	data   interface{}
	result string
}

var (
	ftests = []*functionTest{
		{"{{ add 2 3 }}", nil, "5"},
		{"{{ to_lower .foo }}", map[string]string{"foo": "BAR"}, "bar"},
		{"{{ to_upper .foo }}", map[string]string{"foo": "bar"}, "BAR"},
		{"{{ join .chars .sep }}", map[string]interface{}{"chars": []string{"a", "b", "c"}, "sep": ","}, "a,b,c"},
		{"{{ to_html .s }}", map[string]string{"s": "<foo\nbar"}, "&lt;foo<br>bar"},
		{"{{ mult 2 1.1 }}", nil, "2.2"},
		{"{{ imult 2 1.1 }}", nil, "2"},
		{"{{ concat \"foo\" \"bar\" }}", nil, "foobar"},
		{"{{ if divisible 5 2 }}1{{ else }}0{{ end }}", nil, "0"},
		{"{{ if divisible 4 2 }}1{{ else }}0{{ end }}", nil, "1"},
	}
)

func TestFunctions(t *testing.T) {
	for _, v := range ftests {
		loader := loaders.MapLoader(map[string][]byte{"template.html": []byte(v.tmpl)})
		tmpl, err := Parse(loader, nil, "template.html")
		if err != nil {
			t.Errorf("error parsing %q: %s", v.tmpl, err)
			continue
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, v.data); err != nil {
			t.Errorf("error executing %q: %s", v.tmpl, err)
			continue
		}
		if buf.String() != v.result {
			t.Errorf("expecting %q executing %q, got %q", v.result, v.tmpl, buf.String())
		}
	}
}
