package template

import (
	"html/template"

	htmltemplate "gnd.la/template/internal/htmltemplate"
)

// These types are aliases for the ones defined in
// html/template. They will be escaped by the template
// engine exactly the same as their html/template counterparts.
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

func (v CSS) String() string                        { return string(v) }
func (v CSS) ContentType() htmltemplate.ContentType { return htmltemplate.ContentTypeCSS }

func (v HTML) String() string                        { return string(v) }
func (v HTML) ContentType() htmltemplate.ContentType { return htmltemplate.ContentTypeHTML }

func (v HTMLAttr) String() string                        { return string(v) }
func (v HTMLAttr) ContentType() htmltemplate.ContentType { return htmltemplate.ContentTypeHTMLAttr }

func (v JS) String() string                        { return string(v) }
func (v JS) ContentType() htmltemplate.ContentType { return htmltemplate.ContentTypeJS }

func (v JSStr) String() string                        { return string(v) }
func (v JSStr) ContentType() htmltemplate.ContentType { return htmltemplate.ContentTypeJSStr }

func (v URL) String() string                        { return string(v) }
func (v URL) ContentType() htmltemplate.ContentType { return htmltemplate.ContentTypeURL }
