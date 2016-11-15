package template

import (
	"bytes"
	"html"
	"html/template"
	"testing"
)

type templateEscapeTest struct {
	tmpl   string
	funcs  map[string]interface{}
	data   interface{}
	result string
}

func runEscapeTests(t testing.TB, tests []templateEscapeTest, beforeCompile func(*Template)) {
	for _, v := range tests {
		tmpl := parseNamedText(t, "template.html", v.tmpl, v.funcs, "text/html", beforeCompile)
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

func TestEscape(t *testing.T) {
	fnText := "<div>{{ f }}</div>"
	fieldtext := "<div>{{ .F }}</div>"
	dotText := "<div>{{ . }}</div>"
	p := "<p>hello</p>"
	escapedResult := "<div>" + html.EscapeString(p) + "</div>"
	nonEscapedResult := "<div><p>hello</p></div>"
	escapeTests := []templateEscapeTest{
		{fnText, map[string]interface{}{"f": func() string { return p }}, nil, escapedResult},
		{fnText, map[string]interface{}{"f": func() template.HTML { return template.HTML(p) }}, nil, nonEscapedResult},
		{fnText, map[string]interface{}{"f": func() HTML { return HTML(p) }}, nil, nonEscapedResult},
		{fieldtext, nil, struct{ F string }{p}, escapedResult},
		{fieldtext, nil, struct{ F template.HTML }{template.HTML(p)}, nonEscapedResult},
		{fieldtext, nil, struct{ F HTML }{HTML(p)}, nonEscapedResult},
		{dotText, nil, p, escapedResult},
		{dotText, nil, template.HTML(p), nonEscapedResult},
		{dotText, nil, HTML(p), nonEscapedResult},
	}
	runEscapeTests(t, escapeTests, nil)
}
