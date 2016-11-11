package template

import (
	"html/template"

	itemplate "gnd.la/internal/template"
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
	htmlEscapeFuncs = convertTemplateFuncMap(itemplate.EscapeFuncMap)
)
