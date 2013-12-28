package assets

import (
	"fmt"
	"html/template"
)

func Render(a Asset) (template.HTML, error) {
	html := a.HTML
	if html == "" {
		tag := a.Tag
		if tag == "" {
			return "", fmt.Errorf("asset %q does not specify HTML nor Tag", a.Name)
		}
		attributes := ""
		if a.Attributes != nil {
			attributes = a.Attributes.String()
		}
		if a.Closed {
			html = fmt.Sprintf("<%s %s></%s>", tag, attributes, tag)
		} else {
			html = fmt.Sprintf("<%s %s>", tag, attributes)
		}
	}
	return Conditional(a.Condition, html), nil
}
