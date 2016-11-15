package template

import (
	"html/template"

	htmltemplate "gnd.la/template/internal/htmltemplate"
)

type (
	CSS      template.CSS
	HTML     template.HTML
	HTMLAttr template.HTMLAttr
	JS       template.JS
	JSStr    template.JSStr
	URL      template.URL
)

var (
	htmlEscapeFuncs = convertTemplateFuncMap(htmltemplate.EscapeFuncMap)
)
