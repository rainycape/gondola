package assets

import (
	"fmt"
	"html/template"
)

func Render(a Asset) (template.HTML, error) {
	html := a.AssetHTML()
	if html == "" {
		tag := a.AssetTag()
		if tag == "" {
			return "", fmt.Errorf("asset of type %T does not specify HTML() nor Tag()", a)
		}
		attributes := ""
		if attr := a.AssetAttributes(); attr != nil {
			attributes = attr.String()
		}
		if a.AssetClosed() {
			html = fmt.Sprintf("<%s %s></%s>", tag, attributes, tag)
		} else {
			html = fmt.Sprintf("<%s %s>", tag, attributes)
		}
	}
	return Conditional(a.AssetCondition(), html), nil
}
