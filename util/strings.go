package util

import (
	"unicode"
)

// UnCamelCase transform a camel-cased string into lowercase
// using sep as the separator between words.
func UnCamelCase(s string, sep string) string {
	var r []rune
	rs := []rune(sep)
	for _, v := range []rune(s) {
		if unicode.IsUpper(v) && r != nil {
			r = append(r, rs...)
		}
		r = append(r, unicode.ToLower(v))
	}
	return string(r)
}
