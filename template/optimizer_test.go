package template

import (
	"bytes"
	"reflect"
	"testing"
)

type optimizerTest struct {
	Text         string
	Instructions []inst
	Result       string
}

func TestOptimizerRemoveEmptyTemplate(t *testing.T) {
	// The optimizer should remove the empty template invocations,
	// merge the two writes and then remove the smaller byte slices
	// since they're not referenced anymore.
	expected := []inst{{op: opWB, val: 0}}
	result := "ab"
	tests := []optimizerTest{
		{`{{ define "t1" }}{{ end }}a{{ template "t1" . }}b`, expected, result},
		{`{{ define "t1" }}{{ end }}a{{ template "t1" .foo "bar" "baz" }}b`, expected, result},
		{`{{ define "t1" }}{{ end }}a{{ template "t1" add 1 2 }}b`, expected, result},
		{`{{ define "t1" }}{{ end }}a{{ template "t1" }}b`, expected, result},
	}
	for _, test := range tests {
		tmpl := parseText(t, test.Text)
		code := tmpl.prog.code["template.html"]
		if !reflect.DeepEqual(code, test.Instructions) {
			var buf1, buf2 bytes.Buffer
			tmpl.prog.dumpTemplate(&buf1, "template.html")
			dumpInstructions(&buf2, tmpl.prog, test.Instructions)
			t.Errorf("error optimizing template %q\n\n, got:%q\n\nexpected %q", test.Text, buf1.String(), buf2.String())
		}
	}
}
