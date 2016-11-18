package template

import (
	"text/template"
)

func (t *Template) Text() *template.Template {
	return t.text
}
