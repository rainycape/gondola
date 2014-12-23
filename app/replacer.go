package app

import (
	"bytes"
	"strconv"
	"strings"
)

type replacement struct {
	start  int
	end    int
	source interface{}
}

type replacer struct {
	pattern      string
	replacements []replacement
}

func newReplacer(pattern string) *replacer {
	var replacements []replacement
	ii := 0
	for ii < len(pattern) {
		p := strings.IndexByte(pattern[ii:], '$')
		if p < 0 {
			break
		}
		p += ii
		if p == len(pattern)-1 {
			// TODO: Error here?
			break
		}
		ch := pattern[p+1]
		if ch != '{' {
			ii++
			if ch == '$' {
				// Escaped
				ii++
			}
			continue
		}
		start := p + 2
		end := strings.IndexByte(pattern[start:], '}')
		if end < 0 {
			break
		}
		end += start
		src := pattern[start:end]
		// Need to remove 2 chr from begin and 1 from end
		repl := replacement{start: start - 2, end: end + 1}
		if n, err := strconv.Atoi(src); err == nil {
			repl.source = n
		} else {
			repl.source = src
		}
		replacements = append(replacements, repl)
		ii = end + 1
	}
	if len(replacements) > 0 {
		return &replacer{pattern: pattern, replacements: replacements}
	}
	return nil
}

func (r *replacer) Replace(provider ContextProvider) string {
	var buf bytes.Buffer
	ii := 0
	for _, v := range r.replacements {
		buf.WriteString(r.pattern[ii:v.start])
		switch x := v.source.(type) {
		case int:
			arg, _ := provider.Arg(x)
			buf.WriteString(arg)
		case string:
			arg, _ := provider.Param(x)
			buf.WriteString(arg)
		default:
			panic("unreachable")
		}
		ii = v.end
	}
	buf.WriteString(r.pattern[ii:])
	return buf.String()
}
