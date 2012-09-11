package util

import (
	"github.com/fiam/gounidecode/unidecode"
	"regexp"
	"strings"
)

var (
	slugRegexp = regexp.MustCompile("\\W+")
)

func Slug(s string) string {
	decoded := unidecode.Unidecode(s)
	spaceless := slugRegexp.ReplaceAllString(decoded, "-")
	return strings.ToLower(strings.Trim(spaceless, "-"))
}
