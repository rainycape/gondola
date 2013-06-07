package util

import (
	"unicode"
)

// UnCamelCase transform a camel-cased string into lowercase
// using sep as the separator between words. Multiple uppercase
// characters together are treated as a single word (e.g. TESTFoo
// returns test_foo).
func UnCamelCase(s string, sep string) string {
	if s == "" {
		return ""
	}
	rs := []rune(sep)
	runes := []rune(s)
	r := []rune{unicode.ToLower(runes[0])}
	last := len(runes) - 1
	if last > 1 {
		for ii, v := range runes[1:last] {
			if unicode.IsUpper(v) && unicode.IsLower(runes[ii+2]) {
				r = append(r, rs...)
			}
			r = append(r, unicode.ToLower(v))
		}
		r = append(r, unicode.ToLower(runes[last]))
	}
	return string(r)
}
