package app

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
)

type argumentCountError struct {
	Count int
	Min   int
	Max   int
}

func (e *argumentCountError) Error() string {
	return fmt.Sprintf("Invalid number of arguments %d. Minimum is %d, maximum is %d.", e.Count, e.Min, e.Max)
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

func walk(r *syntax.Regexp, f func(r *syntax.Regexp) bool) bool {
	stop := f(r)
	if !stop {
		for _, v := range r.Sub {
			if walk(v, f) {
				stop = true
				break
			}
		}
	}
	if !stop {
		stop = f(nil)
	}
	return stop
}

func compileToSyntaxRegexp(r *regexp.Regexp) *syntax.Regexp {
	re, _ := syntax.Parse(r.String(), syntax.Perl)
	return re
}

func formatRegexp(r *regexp.Regexp, args []interface{}) (string, error) {
	re := compileToSyntaxRegexp(r)
	max := re.MaxCap()
	min := minCap(re)
	rem := len(args)
	if rem < min || rem > max {
		return "", &argumentCountError{rem, min, max}
	}
	var formatted []string
	var err error
	var stack []*syntax.Regexp
	walk(re, func(r *syntax.Regexp) bool {
		if r == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		stack = append(stack, r)
		switch r.Op {
		case syntax.OpLiteral:
			// Check if this literal was already provided by a previous
			// replacement. If we're inside a capture group, the provided
			// replacement must have already satisfied this literal, otherwise
			// it would have failed.
			provided := false
			for _, v := range stack {
				if v.Op == syntax.OpCapture {
					provided = true
					break
				}
			}
			if !provided {
				formatted = append(formatted, string(r.Rune))
			}
			if rem == 0 {
				return true
			}
		case syntax.OpQuest:
			if rem == 0 {
				return true
			}
		case syntax.OpCapture:
			c := r.Cap
			if rem == 0 || c > len(args) {
				return true
			}
			cur := fmt.Sprintf("%v", args[c-1])
			patt := r.String()
			if matched, _ := regexp.MatchString(patt, cur); !matched {
				err = fmt.Errorf("Invalid replacement at index %d. Format is %q, replacement is %q.", c-1, patt, cur)
				return true
			}
			formatted = append(formatted, cur)
			rem--
		}
		return false
	})
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
