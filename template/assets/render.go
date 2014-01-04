package assets

import (
	"fmt"
	"html/template"
	"io"
)

func Render(m *Manager, a *Asset) (template.HTML, error) {
	var html string
	switch a.Type {
	case TypeCSS:
		if a.Attributes != nil {
			html = fmt.Sprintf("<link %s rel=\"stylesheet\" type=\"text/css\" href=\"%s\">", a.Attributes.String(), m.URL(a.Name))
		} else {
			html = fmt.Sprintf("<link rel=\"stylesheet\" type=\"text/css\" href=\"%s\">", m.URL(a.Name))
		}
	case TypeJavascript:
		if a.Attributes != nil {
			html = fmt.Sprintf("<script %s type=\"text/javascript\" src=\"%s\"></script>", a.Attributes.String(), m.URL(a.Name))
		} else {
			html = fmt.Sprintf("<script type=\"text/javascript\" src=\"%s\"></script>", m.URL(a.Name))
		}
	default:
		if a.HTML == "" {
			return "", fmt.Errorf("asset %q of Other type must specify HTML", a.Name)
		}
		html = a.HTML
	}
	return Conditional(a.Condition, html), nil
}

func RenderTo(w io.Writer, m *Manager, a *Asset) error {
	h, err := Render(m, a)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, string(h))
	return err
}
