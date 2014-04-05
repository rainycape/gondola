package stringutil

import (
	"unicode/utf8"
)

func Reverse(s string) string {
	p := utf8.RuneCountInString(s)
	out := make([]rune, p)
	for _, c := range s {
		p--
		out[p] = c
	}
	return string(out)
}
