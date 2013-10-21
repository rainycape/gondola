package assets

import (
	"fmt"
	"html/template"
	"regexp"
	"strconv"
)

var (
	conditionalRe = regexp.MustCompile("(?i:^!?ie((?:l|g)te?)?(\\d+)?$)")
)

type Comparison int

type Condition struct {
	Comparison Comparison
	Version    int
}

const (
	ComparisonNone Comparison = iota
	ComparisonIf
	ComparisonIfNot
	ComparisonIfEqual
	ComparisonIfLessThan
	ComparisonIfLessThanOrEqual
	ComparisonIfGreaterThan
	ComparisonIfGreaterThanOrEqual
)

var comparisons = map[Comparison]string{
	ComparisonIfEqual:              "",
	ComparisonIfLessThan:           "lt ",
	ComparisonIfLessThanOrEqual:    "lte ",
	ComparisonIfGreaterThan:        "gt ",
	ComparisonIfGreaterThanOrEqual: "gte ",
}

func ParseCondition(val string) (*Condition, error) {
	if val == "" {
		return nil, nil
	}
	if conditionalRe.MatchString(val) {
		m := conditionalRe.FindStringSubmatch(val)
		var version int
		var err error
		if len(m) == 3 {
			if v := m[len(m)-1]; v != "" {
				version, err = strconv.Atoi(m[len(m)-1])
				if err != nil {
					return nil, err
				}
			}
			var cmp Comparison
			switch m[1] {
			case "lt":
				cmp = ComparisonIfLessThan
			case "lte":
				cmp = ComparisonIfLessThanOrEqual
			case "gt":
				cmp = ComparisonIfGreaterThan
			case "gte":
				cmp = ComparisonIfGreaterThanOrEqual
			case "":
				cmp = ComparisonIfEqual
			default:
				return nil, fmt.Errorf("invalid condition %q", val)
			}
			return &Condition{
				Comparison: cmp,
				Version:    version,
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid condition %q", val)
}

func Conditional(c *Condition, html string) template.HTML {
	if c == nil {
		return template.HTML(html)
	}
	var conditional string
	var cmp string
	var vers string
	switch c.Comparison {
	case ComparisonNone:
		conditional = html
	case ComparisonIfNot:
		conditional = fmt.Sprintf("<!--[if !IE]> -->%s<!-- <![endif]-->", html)
	default:
		cmp = comparisons[c.Comparison]
		if c.Version != 0 {
			vers = fmt.Sprintf(" %d", c.Version)
		}
		fallthrough
	case ComparisonIf:
		conditional = fmt.Sprintf("<!--[if %sIE%s]>%s<![endif]-->", cmp, vers, html)
	}
	return template.HTML(conditional)
}
