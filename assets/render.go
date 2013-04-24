package assets

import (
	"fmt"
	"html/template"
)

func Render(a Asset) (template.HTML, error) {
	html := a.HTML()
	if html == "" {
		tag := a.Tag()
		if tag == "" {
			return "", fmt.Errorf("Asset of type %T does not specify HTML() nor Tag()", a)
		}
		attributes := ""
		if attr := a.Attributes(); attr != nil {
			attributes = attr.String()
		}
		if a.Closed() {
			html = fmt.Sprintf("<%s %s></%s>", tag, attributes, tag)
		} else {
			html = fmt.Sprintf("<%s %s>", tag, attributes)
		}
	}
	return Conditional(a.Condition(), a.ConditionVersion(), html), nil
}
