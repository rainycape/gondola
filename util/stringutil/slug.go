package stringutil

import (
	"regexp"
	"strings"

	"github.com/rainycape/unidecode"
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
	decoded := unidecode.Unidecode(s)
	spaceless := slugRegexp.ReplaceAllString(decoded, "-")
	return strings.ToLower(strings.Trim(spaceless, "-"))
}

// SlugN works like Slug, but returns at string with, at
// most n characters. If n is <= 0, it works exactly
// like Slug.
func SlugN(s string, n int) string {
	slug := Slug(s)
	if n > 0 && len(slug) > n {
		slug = strings.Trim(slug[:n], "-")
	}
	return slug
}
