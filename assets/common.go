package assets

import (
	"fmt"
	"html/template"
	"strings"
)

type Attributes map[string]string

func (a Attributes) String() string {
	var attrs []string
	for k, v := range map[string]string(a) {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", k, strings.Replace(v, "\"", "\\\"", -1)))
	}
	return strings.Join(attrs, " ")
}

type CommonAsset struct {
	Manager          Manager
	Name             string
	TagName          string
	MustClose        bool
	Attributes       Attributes
	Type             CodeType
	Condition        Condition
	ConditionVersion int
}

func (c *CommonAsset) CodeType() CodeType {
	return c.Type
}

func (c *CommonAsset) HTML() template.HTML {
	html := ""
	if c.TagName != "" {
		attributes := ""
		if c.Attributes != nil {
			attributes = c.Attributes.String()
		}
		if c.MustClose {
			html = fmt.Sprintf("<%s %s></%s>", c.TagName, attributes, c.TagName)
		} else {
			html = fmt.Sprintf("<%s %s>", c.TagName, attributes)
		}
	}
	return Conditional(c.Condition, c.ConditionVersion, html)
}

func ParseCommonAssets(m Manager, names []string, options map[string]string) ([]*CommonAsset, error) {
	cond := ConditionNone
	vers := 0
	if ifopt := options["if"]; ifopt != "" {
		var err error
		cond, vers, err = ParseCondition(ifopt)
		if err != nil {
			return nil, err
		}
		delete(options, "if")
	}
	common := make([]*CommonAsset, len(names))
	for ii, v := range names {
		common[ii] = &CommonAsset{
			Manager:          m,
			Name:             v,
			Condition:        cond,
			ConditionVersion: vers,
		}
	}
	return common, nil
}
