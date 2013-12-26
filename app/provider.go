package app

import (
	"regexp"
)

// ContextProvider represents the interface which
// types which provide a context with its arguments
// and parameters must satisfy.
type ContextProvider interface {
	// Count returns the number of arguments
	Count() int
	// Arg returns the argument at index idx or the
	// empty string if there's no such argument.
	Arg(idx int) string
	// Param returns the parameter named with the given
	// name or the empty string if there's no such parameter.
	Param(name string) string
}

type regexpProvider struct {
	re        *regexp.Regexp
	path      string
	matches   []int
	arguments []string
}

func (r *regexpProvider) buildArguments() {
	n := r.re.NumSubexp() + 1
	m := r.matches
	for ii := 0; ii < n; ii++ {
		if x := 2 * ii; x < len(m) && m[x] >= 0 {
			r.arguments = append(r.arguments, r.path[m[x]:m[x+1]])
		}
	}
}

func (r *regexpProvider) Count() int {
	if len(r.arguments) == 0 {
		r.buildArguments()
	}
	return len(r.arguments) - 1
}

func (r *regexpProvider) Arg(idx int) string {
	if len(r.arguments) == 0 {
		r.buildArguments()
	}
	if idx < len(r.arguments)-1 {
		return r.arguments[idx+1]
	}
	return ""
}

func (r *regexpProvider) Param(name string) string {
	if len(r.arguments) == 0 {
		r.buildArguments()
	}
	for ii, v := range r.re.SubexpNames() {
		if v == name {
			if ii < len(r.arguments) {
				return r.arguments[ii]
			}
			break
		}
	}
	return ""
}

func (r *regexpProvider) reset(re *regexp.Regexp, path string, matches []int) {
	r.re = re
	r.path = path
	r.matches = matches
	r.arguments = r.arguments[:0]
}
