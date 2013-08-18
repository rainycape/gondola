package mux

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
)

type ArgumentCountError struct {
	MinArguments int
	MaxArguments int
}

func (e *ArgumentCountError) Error() string {
	return fmt.Sprintf("Invalid number of arguments. Minimum is %d, maximum is %d.", e.MinArguments, e.MaxArguments)
}

func minCap(re *syntax.Regexp) int {
	c := 0
	if re.Op == syntax.OpCapture {
		c++
	}
	for _, v := range re.Sub {
		if v.Op == syntax.OpStar || v.Op == syntax.OpQuest || (v.Op == syntax.OpRepeat && v.Min == 0) {
			continue
		}
		c += minCap(v)
	}
	return c
}

func walk(r *syntax.Regexp, f func(*syntax.Regexp) bool) bool {
	stop := f(r)
	if !stop {
		for _, v := range r.Sub {
			if walk(v, f) {
				stop = true
				break
			}
		}
	}
	return stop
}

func FormatRegexp(r *regexp.Regexp, strict bool, args ...interface{}) (string, error) {
	re, _ := syntax.Parse(r.String(), syntax.Perl)
	max := re.MaxCap()
	min := minCap(re)
	l := len(args)
	if l < min || l > max {
		return "", &ArgumentCountError{min, max}
	}
	var formatted []string
	var err error
	if l == 0 {
		walk(re, func(r *syntax.Regexp) bool {
			switch r.Op {
			case syntax.OpLiteral:
				formatted = append(formatted, string(r.Rune))
			case syntax.OpCapture:
				return true
			}
			return false
		})
	} else {
		stop := false
		walk(re, func(r *syntax.Regexp) bool {
			switch r.Op {
			case syntax.OpLiteral:
				formatted = append(formatted, string(r.Rune))
				if stop {
					return true
				}
			case syntax.OpCapture:
				c := r.Cap
				cur := fmt.Sprintf("%v", args[c-1])
				if strict {
					patt := r.String()
					if matched, _ := regexp.MatchString(patt, cur); !matched {
						err = fmt.Errorf("Invalid replacement at index %d. Format is %q, replacement is %q.", c-1, patt, cur)
						return true
					}
				}
				formatted = append(formatted, cur)
				stop = c == l
			}
			return false
		})
	}
	if err != nil {
		return "", err
	}
	return strings.Join(formatted, ""), nil
}

func literalRegexp(r *regexp.Regexp) string {
	re, _ := syntax.Parse(r.String(), syntax.Perl)
	if re.MaxCap() == 0 && re.Op == syntax.OpConcat && len(re.Sub) == 3 &&
		re.Sub[0].Op == syntax.OpBeginText &&
		re.Sub[1].Op == syntax.OpLiteral &&
		re.Sub[2].Op == syntax.OpEndText {

		return string(re.Sub[1].Rune)
	}
	return ""
}
