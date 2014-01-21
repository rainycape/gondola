package template

import (
	"bytes"
	"gnd.la/loaders"
	"gnd.la/template/assets"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"
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
		{"{{ \"output\" | printf \"%s\" }}", nil, "output"},
		{"{{ call .foo }}", map[string]interface{}{"foo": func() string { return "bar" }}, "bar"},
		{"{{ .Foo }}", struct{ Foo string }{"bar"}, "bar"},
		{"{{ .Foo }}", &testType{}, "bar"},
		{"{{ .Bar \"this\" }}", &testType{}, "bared-this"},
		{"{{ .t.Bar .foo }}", map[string]interface{}{"t": &testType{}, "foo": "foo"}, "bared-foo"},
		{"{{ .t.Bar (concat .foo \"bar\") }}", map[string]interface{}{"t": &testType{}, "foo": "foo"}, "bared-foobar"},
		{"{{ with .A }}{{ . }}{{ else }}no{{ end }}", map[string]string{"A": "yes"}, "yes"},
		{"{{ with .A }}{{ . }}{{ else }}no{{ end }}", nil, "no"},
		{"{{ with .A }}{{ . }}{{ end }}", nil, ""},
		{"{{ range . }}{{ . }}{{ end }}", []int{1, 2, 3}, "123"},
		{"{{ range . }}{{ . }}{{ end }}{{ . }}", []int{1, 2, 3}, "123[1 2 3]"},
		{"{{ range $idx, $el := . }}{{ $idx }}{{ $el }}{{ end }}", []int{1, 2, 3}, "011223"},
		{"{{ range $el := . }}{{ $el }}{{ end }}", []int{1, 2, 3}, "123"},
		{"{{ range $el := . }}{{ . }}{{ end }}", []int{1, 2, 3}, "123"},
		{"{{ range $idx, $el := . }}{{ . }}{{ end }}", []int{1, 2, 3}, "123"},
		{"{{ range . }}{{ else }}nope{{ end }}", nil, "nope"},
		{"{{ range $k, $v := . }}{{ $k }}={{ $v }}{{ end }}", map[string]int{"b": 2, "c": 3, "a": 1}, "a=1b=2c=3"},
		{"{{ range . }}{{ range . }}{{ if even . }}{{ . }}{{ end }}{{ end }}{{ end }}", [][]int{[]int{1, 2, 3, 4, 5, 6}}, "246"},
		{"{{ define \"a\" }}a{{ end }}{{ range . }}{{ template \"a\" . }}{{ end }}", []int{1, 2, 3}, "aaa"},
		{"{{ define \"a\" }}a{{ . }}{{ . }}{{ end }}{{ range . }}{{ template \"a\" . }}{{ end }}", []int{1, 2, 3}, "a11a22a33"},
		{"{{ define \"a\" }}a{{ . }}{{ . }}{{ end }}{{ if . }}{{ template \"a\" . }}{{ end }}", 0, ""},
		{"{{ define \"a\" }}a{{ . }}{{ . }}{{ end }}{{ if . }}{{ template \"a\" . }}{{ end }}", 1, "a11"},
	}
	compilerErrorTests = []*templateTest{
		{"{{ range . }}{{ else }}nope{{ end }}", 5, "template.html:1:9: can't range over int"},
		{"{{ . }}\n{{ range . }}{{ else }}nope{{ end }}", 5, "template.html:2:9: can't range over int"},
		{"{{ . }}\n{{ range .foo }}{{ else }}nope{{ end }}\n{{ range .bar }}{{ . }}{{ end }} ", map[string]interface{}{"foo": []int{}, "bar": ""}, "template.html:3:9: can't range over string"},
	}
)

func parseText(tb testing.TB, text string) *Template {
	loader := loaders.MapLoader(map[string][]byte{"template.html": []byte(text)})
	tmpl, err := Parse(loader, nil, "template.html")
	if err != nil {
		tb.Errorf("error parsing %q: %s", text, err)
		return nil
	}
	if err := tmpl.Compile(); err != nil {
		tb.Errorf("error compiling %q: %s", text, err)
		return nil
	}
	return tmpl
}

func parseTestTemplate(tb testing.TB, name string) *Template {
	loader := loaders.FSLoader("_testdata")
	tmpl := New(loader, assets.NewManager(loader, ""))
	tmpl.Funcs(FuncMap{"t": func(s string) string { return s }})
	if err := tmpl.Parse(name); err != nil {
		tb.Errorf("error parsing %q: %s", name, err)
		return nil
	}
	if err := tmpl.Compile(); err != nil {
		tb.Errorf("error compiling %q: %s", name, err)
		return nil
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
	tests = append(tests, compilerTests...)
	for _, v := range tests {
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

func TestCompilerErrors(t *testing.T) {
	for _, v := range compilerErrorTests {
		tmpl := parseText(t, v.tmpl)
		if tmpl == nil {
			continue
		}
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, v.data)
		if err == nil {
			t.Errorf("expecting an error when executing %q, got nil", v.tmpl)
			continue
		}
		if err.Error() != v.result {
			t.Logf("template is %q", v.tmpl)
			t.Errorf("expecting error %q, got %q", v.result, err.Error())
		}
	}
}

func TestBigTemplate(t *testing.T) {
	const name = "1.html"
	tmpl := parseTestTemplate(t, name)
	if tmpl != nil {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, nil); err != nil {
			t.Errorf("error executing template %s: %s", name, err)
		}
	}
}

func benchmarkTests() []*templateTest {
	var tests []*templateTest
	tests = append(tests, ftests...)
	tests = append(tests, compilerTests...)
	return tests
}

func benchmarkTemplate(b *testing.B, tests []*templateTest) {
	b.ReportAllocs()
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
		}
		buf.Reset()
	}
}

func BenchmarkTemplate(b *testing.B) {
	benchmarkTemplate(b, benchmarkTests())
}

func BenchmarkHTMLTemplate(b *testing.B) {
	b.ReportAllocs()
	tests := benchmarkTests()
	templates := make([]*template.Template, len(tests))
	for ii, v := range tests {
		tmpl := template.New("template.html")
		tmpl.Funcs(template.FuncMap(templateFuncs))
		_, err := tmpl.Parse(v.tmpl)
		if err != nil {
			b.Fatalf("can't parse %q: %s", v.tmpl, err)
		}
		// Execute once to add the escaping hooks
		tmpl.Execute(ioutil.Discard, nil)
		templates[ii] = tmpl
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for ii, v := range templates {
			v.Execute(&buf, tests[ii].data)
		}
		buf.Reset()
	}
}

func BenchmarkBig(b *testing.B) {
	b.ReportAllocs()
	const name = "1.html"
	tmpl := parseTestTemplate(b, name)
	if tmpl == nil {
		return
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		tmpl.Execute(&buf, nil)
		buf.Reset()
	}
}

func BenchmarkHTMLBig(b *testing.B) {
	b.ReportAllocs()
	tmpl := template.New("")
	tmpl.Funcs(template.FuncMap{"t": func(s string) string { return s }})
	tmpl.Funcs(template.FuncMap(templateFuncs))
	readFile := func(name string) string {
		data, err := ioutil.ReadFile(filepath.Join("_testdata", name))
		if err != nil {
			b.Fatal(err)
		}
		return "{{ $Vars := .Vars }}\n" + string(data)
	}
	if _, err := tmpl.Parse(readFile("1.html")); err != nil {
		b.Fatal(err)
	}
	t2 := tmpl.New("2.html")
	if _, err := t2.Parse(readFile("2.html")); err != nil {
		b.Fatal(err)
	}
	if err := tmpl.Execute(ioutil.Discard, nil); err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		tmpl.Execute(&buf, nil)
		buf.Reset()
	}
}

func BenchmarkRange(b *testing.B) {
	var tests []*templateTest
	for _, v := range benchmarkTests() {
		if strings.Contains(v.tmpl, "range") {
			tests = append(tests, v)
		}
	}
	benchmarkTemplate(b, tests)
}
