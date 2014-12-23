package app

import "regexp"

// ContextProvider represents the interface used
// by Context to retrieve the parameters received.
// Providers
// types which provide a context with its arguments
// and parameters must satisfy.
type ContextProvider interface {
	// Count returns total number of arguments
	Count() int
	// Arg returns the argument at index idx or the
	// empty string if there's no such argument.
	// The last return value is used to disambiguate if
	// the parameter was provided but empty or not
	// provided at all.
	Arg(idx int) (string, bool)
	// Param returns the parameter named with the given
	// name or the empty string if there's no such parameter.
	// The last return value is used to disambiguate if
	// the parameter was provided but empty or not
	// provided at all.
	Param(name string) (string, bool)
	// ParamNames returns the names of the available parameters,
	// in the order they were specified. Non-named parameters
	// must be present in the returned slice as a empty string.
	ParamNames() []string
}

type regexpProvider struct {
	re          *regexp.Regexp
	path        string
	matches     []int
	arguments   []string
	notProvided map[int]bool
	params      map[string]string
}

func (r *regexpProvider) buildArguments() {
	if len(r.arguments) > 0 {
		return
	}
	n := r.re.NumSubexp() + 1
	m := r.matches
	names := r.re.SubexpNames()
	for ii := 0; ii < n; ii++ {
		var arg string
		if x := 2 * ii; x < len(m) && m[x] >= 0 {
			arg = r.path[m[x]:m[x+1]]
			if name := names[ii]; name != "" {
				r.params[name] = arg
			}
		} else {
			r.notProvided[ii] = true
		}
		r.arguments = append(r.arguments, arg)
	}
}

func (r *regexpProvider) Count() int {
	return len(r.arguments) - 1
}

func (r *regexpProvider) Arg(idx int) (string, bool) {
	if idx < len(r.arguments)-1 {
		return r.arguments[idx+1], !r.notProvided[idx+1]
	}
	return "", false
}

func (r *regexpProvider) Param(name string) (string, bool) {
	val, found := r.params[name]
	return val, found
}

func (r *regexpProvider) ParamNames() []string {
	return r.re.SubexpNames()[1:]
}

func (r *regexpProvider) reset(re *regexp.Regexp, path string, matches []int) {
	r.re = re
	r.path = path
	r.matches = matches
	r.arguments = r.arguments[:0]
	r.notProvided = make(map[int]bool)
	r.params = make(map[string]string)
	r.buildArguments()
}
