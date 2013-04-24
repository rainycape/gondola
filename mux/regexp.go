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

func FormatRegexp(r *regexp.Regexp, args ...interface{}) (string, error) {
	re, _ := syntax.Parse(r.String(), syntax.Perl)
	max := re.MaxCap()
	min := minCap(re)
	l := len(args)
	if l < min || l > max {
		return "", &ArgumentCountError{min, max}
	}
	var formatted []string
	var err error
	stop := false
	walk(re, func(r *syntax.Regexp) bool {
		if r.Op == syntax.OpLiteral {
			formatted = append(formatted, string(r.Rune))
			if stop {
				return true
			}
		} else if r.Op == syntax.OpCapture {
			if l == 0 {
				return true
			}
			c := r.Cap
			// TODO: Check if arguments match pattern?
			formatted = append(formatted, fmt.Sprintf("%v", args[c-1]))
			if c == l {
				stop = true
			}
		}
		return false
	})
	if err != nil {
		return "", err
	}
	return strings.Join(formatted, ""), nil
}
