package template

import (
	itemplate "gnd.la/internal/template"
	"html/template"
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
	htmlEscapeFuncs = FuncMap(itemplate.EscapeFuncMap)
)
