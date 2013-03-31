package util

import (
	"regexp"
	"strings"
)

var (
	slugRegexp = regexp.MustCompile("\\W+")
)

// Slug returns a slugified version of the given string, which
// consists in transliterating unicode characters to ascii
// (e.g. ó becomes o and â becomes a), replacing all sequences of
// whitespaces with '-' and converting to lowercase. Very useful
// for making arbitrary strings, like a post title, part of URLs.
func Slug(s string) string {
	decoded := Unidecode(s)
	spaceless := slugRegexp.ReplaceAllString(decoded, "-")
	return strings.ToLower(strings.Trim(spaceless, "-"))
}
