package assets

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

type Comparison int

type Condition struct {
	Comparison Comparison
	Version    int
	NonIE      bool
}

const (
	ComparisonNone Comparison = iota
	ComparisonEqual
	ComparisonLessThan
	ComparisonLessThanOrEqual
	ComparisonGreaterThan
	ComparisonGreaterThanOrEqual
)

var comparisons = map[Comparison]string{
	ComparisonEqual:              "",
	ComparisonLessThan:           "lt ",
	ComparisonLessThanOrEqual:    "lte ",
	ComparisonGreaterThan:        "gt ",
	ComparisonGreaterThanOrEqual: "gte ",
}

func ParseCondition(val string) (*Condition, error) {
	if val == "" {
		return nil, nil
	}
	parts := strings.Split(val, "-")
	if len(parts) <= 3 && (parts[0] == "ie" || parts[0] == "~ie") {
		var version int
		var err error
		versionIdx := -1
		cmpIdx := -1
		switch len(parts) {
		case 2:
			versionIdx = 1
		case 3:
			cmpIdx = 1
			versionIdx = 2
		}
		cmp := ComparisonEqual
		if versionIdx > 0 {
			version, err = strconv.Atoi(parts[versionIdx])
			if err != nil {
				return nil, fmt.Errorf("invalid version number %q: %s", parts[versionIdx], err)
			}
		}
		if cmpIdx > 0 {
			switch parts[cmpIdx] {
			case "lt":
				cmp = ComparisonLessThan
			case "lte":
				cmp = ComparisonLessThanOrEqual
			case "gt":
				cmp = ComparisonGreaterThan
			case "gte":
				cmp = ComparisonGreaterThanOrEqual
			case "eq":
				cmp = ComparisonEqual
			default:
				return nil, fmt.Errorf("invalid comparison %q", val)
			}
		}
		return &Condition{
			Comparison: cmp,
			Version:    version,
			NonIE:      parts[0][0] == '~',
		}, nil
	}
	return nil, fmt.Errorf("invalid condition %q", val)
}

func Conditional(c *Condition, html string) template.HTML {
	if c == nil || c.Comparison == ComparisonNone {
		return template.HTML(html)
	}
	var conditional string
	var vers string
	cmp := comparisons[c.Comparison]
	if c.Version != 0 {
		vers = fmt.Sprintf(" %d", c.Version)
	}
	if c.NonIE {
		conditional = fmt.Sprintf("<!--[if %sIE%s]><!-->%s<!--<![endif]-->", cmp, vers, html)
	} else {
		conditional = fmt.Sprintf("<!--[if %sIE%s]>%s<![endif]-->", cmp, vers, html)
	}
	return template.HTML(conditional)
}
