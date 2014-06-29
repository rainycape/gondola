package app

import (
	"bytes"
	"fmt"
	"regexp"
	"regexp/syntax"
	"sync"
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

type regexpCache struct {
	re    *syntax.Regexp
	max   int
	min   int
	cache map[string]*regexp.Regexp
	mu    sync.RWMutex
}

func newRegexpCache(r *regexp.Regexp) *regexpCache {
	s := r.String()
	re, _ := syntax.Parse(s, syntax.Perl)
	return &regexpCache{
		re:    re,
		min:   minCap(re),
		max:   re.MaxCap(),
		cache: make(map[string]*regexp.Regexp),
	}
}

func formatRegexp(re *regexpCache, args []interface{}) (string, error) {
	rem := len(args)
	if rem < re.min || rem > re.max {
		return "", &argumentCountError{rem, re.min, re.max}
	}
	var buf bytes.Buffer
	var err error
	var stack []*syntax.Regexp
	walk(re.re, func(r *syntax.Regexp) bool {
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
				for _, ru := range r.Rune {
					buf.WriteRune(ru)
				}
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
			re.mu.RLock()
			sr, ok := re.cache[patt]
			re.mu.RUnlock()
			if !ok {
				sr = regexp.MustCompile(patt)
				re.mu.Lock()
				re.cache[patt] = sr
				re.mu.Unlock()
			}
			if !sr.MatchString(cur) {
				err = fmt.Errorf("Invalid replacement at index %d. Format is %q, replacement is %q.", c-1, patt, cur)
				return true
			}
			buf.WriteString(cur)
			rem--
		}
		return false
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
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
