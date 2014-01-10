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
		{"{{ concat (concat \"foo\" \"bar\") \"baz\" }}", nil, "foobarbaz"},
		{"{{ if divisible 5 2 }}1{{ else }}0{{ end }}", nil, "0"},
		{"{{ if divisible 4 2 }}1{{ else }}0{{ end }}", nil, "1"},
	}
)

func parseText(tb testing.TB, text string) *Template {
	loader := loaders.MapLoader(map[string][]byte{"template.html": []byte(text)})
	tmpl, err := Parse(loader, nil, "template.html")
	if err != nil {
		tb.Errorf("error parsing %q: %s", text, err)
	}
	return tmpl
}

func TestFunctions(t *testing.T) {
	for _, v := range ftests {
		tmpl := parseText(t, v.tmpl)
		if tmpl == nil {
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

func TestCompiler(t *testing.T) {
	for _, v := range ftests {
		tmpl := parseText(t, v.tmpl)
		if tmpl == nil {
			continue
		}
		pr, err := NewProgram(tmpl)
		if err != nil {
			t.Errorf("error compiling %q: %s", v.tmpl, err)
			continue
		}
		var buf bytes.Buffer
		if err := pr.Execute(&buf, v.data); err != nil {
			t.Errorf("error executing %q: %s", v.tmpl, err)
			continue
		}
		if buf.String() != v.result {
			t.Errorf("expecting %q executing %q, got %q", v.result, v.tmpl, buf.String())
		}
	}
}

func BenchmarkExecute(b *testing.B) {
	b.ReportAllocs()
	templates := make([]*Template, len(ftests))
	for ii, v := range ftests {
		tmpl := parseText(b, v.tmpl)
		if tmpl == nil {
			b.Fatalf("can't parse %q", v.tmpl)
		}
		templates[ii] = tmpl
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for ii, v := range templates {
			v.Execute(&buf, ftests[ii].data)
			buf.Reset()
		}
	}
}

func BenchmarkExecuteProgram(b *testing.B) {
	b.ReportAllocs()
	programs := make([]*Program, len(ftests))
	for ii, v := range ftests {
		tmpl := parseText(b, v.tmpl)
		if tmpl == nil {
			b.Fatalf("can't parse %q", v.tmpl)
		}
		pr, err := NewProgram(tmpl)
		if err != nil {
			b.Fatalf("can't compile %", v.tmpl)
		}
		programs[ii] = pr
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for ii, v := range programs {
			v.Execute(&buf, ftests[ii].data)
			buf.Reset()
		}
	}
}
