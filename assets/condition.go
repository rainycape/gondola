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

type Condition int

const (
	ConditionNone Condition = iota
	ConditionIf
	ConditionIfNot
	ConditionIfEqual
	ConditionIfLessThan
	ConditionIfLessThanOrEqual
	ConditionIfGreaterThan
	ConditionIfGreaterThanOrEqual
)

var conditions = map[Condition]string{
	ConditionIfEqual:              "",
	ConditionIfLessThan:           "lt ",
	ConditionIfLessThanOrEqual:    "lte ",
	ConditionIfGreaterThan:        "gt ",
	ConditionIfGreaterThanOrEqual: "gte ",
}

func ParseCondition(val string) (Condition, int, error) {
	if val == "" {
		return ConditionNone, 0, nil
	}
	if conditionalRe.MatchString(val) {
		m := conditionalRe.FindStringSubmatch(val)
		var version int
		var err error
		if len(m) == 3 {
			if v := m[len(m)-1]; v != "" {
				version, err = strconv.Atoi(m[len(m)-1])
				if err != nil {
					return ConditionNone, 0, err
				}
			}
			var cond Condition
			switch m[1] {
			case "lt":
				cond = ConditionIfLessThan
			case "lte":
				cond = ConditionIfLessThanOrEqual
			case "gt":
				cond = ConditionIfGreaterThan
			case "gte":
				cond = ConditionIfGreaterThanOrEqual
			case "":
				cond = ConditionIfEqual
			default:
				return ConditionNone, 0, fmt.Errorf("Invalid condition %q", val)
			}
			return cond, version, nil
		}
	}
	return ConditionNone, 0, fmt.Errorf("Invalid condition %q", val)
}

func Conditional(cond Condition, version int, html string) template.HTML {
	var conditional string
	var cmp string
	var vers string
	switch cond {
	case ConditionNone:
		conditional = html
	case ConditionIfNot:
		conditional = fmt.Sprintf("<!--[if !IE]> -->%s<!-- <![endif]-->", html)
	default:
		cmp = conditions[cond]
		if version != 0 {
			vers = fmt.Sprintf(" %d", version)
		}
		fallthrough
	case ConditionIf:
		conditional = fmt.Sprintf("<!--[if %sIE%s]>%s<![endif]-->", cmp, vers, html)
	}
	return template.HTML(conditional)
}
