package template

import (
	"bytes"
	"gnd.la/loaders"
	"gnd.la/template/assets"
	"testing"
)

type templateTest struct {
	tmpl   string
	data   interface{}
	result string
}

type testType struct {
}

func (t *testType) Foo() string {
	return "bar"
}

func (t *testType) Bar(s string) string {
	return "bared-" + s
}

var (
	ftests = []*templateTest{
		{"{{ $one := 1 }}{{ $two := 2 }}{{ $three := 3 }}{{ $one }}+{{ $two }}+{{ $three }}={{ add $one $two $three }}", nil, "1+2+3=6"},
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
	compilerTests = []*templateTest{
		{"{{ call .foo }}", map[string]interface{}{"foo": func() string { return "bar" }}, "bar"},
		{"{{ .Foo }}", struct{ Foo string }{"bar"}, "bar"},
		{"{{ .Foo }}", &testType{}, "bar"},
		{"{{ .Bar \"this\" }}", &testType{}, "bared-this"},
		{"{{ .t.Bar .foo }}", map[string]interface{}{"t": &testType{}, "foo": "foo"}, "bared-foo"},
		{"{{ .t.Bar (concat .foo \"bar\") }}", map[string]interface{}{"t": &testType{}, "foo": "foo"}, "bared-foobar"},
		{"{{ with .A }}{{ . }}{{ else }}no{{ end }}", map[string]string{"A": "yes"}, "yes"},
		{"{{ with .A }}{{ . }}{{ else }}no{{ end }}", nil, "no"},
		{"{{ with .A }}{{ . }}{{ end }}", nil, ""},
		/*{"{{ range . }}{{ . }}{{ end }}", []int{1, 2, 3}, "123"},
		{"{{ range $idx, $el := . }}{{ $idx }}{{ $el }}{{ end }}", []int{1, 2, 3}, "011223"},
		{"{{ range $el := . }}{{ $el }}{{ end }}", []int{1, 2, 3}, "123"},
		{"{{ range . }}{{ else }}nope{{ end }}", nil, "nope"},*/
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
	var tests []*templateTest
	tests = append(tests, ftests...)
	tests = append(tests, compilerTests...)
	for _, v := range tests {
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

func benchmarkTests() []*templateTest {
	var tests []*templateTest
	tests = append(tests, ftests...)
	tests = append(tests, compilerTests...)
	return tests
}

func BenchmarkExecute(b *testing.B) {
	b.ReportAllocs()
	tests := benchmarkTests()
	templates := make([]*Template, len(tests))
	for ii, v := range tests {
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
			v.Execute(&buf, tests[ii].data)
			buf.Reset()
		}
	}
}

func BenchmarkExecuteProgram(b *testing.B) {
	b.ReportAllocs()
	tests := benchmarkTests()
	programs := make([]*Program, len(tests))
	for ii, v := range tests {
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
			v.Execute(&buf, tests[ii].data)
			buf.Reset()
		}
	}
}
