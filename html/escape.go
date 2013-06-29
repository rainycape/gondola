package html

import (
	"html"
)

func Escape(s string) string {
	return html.EscapeString(s)
}

func Unescape(s string) string {
	return html.UnescapeString(s)
}
