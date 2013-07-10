package astutil

import (
	"strings"
)

func unquote(s string) string {
	s = strings.Trim(s, "\"")
	return strings.Replace(s, "\\\"", "\"", -1)
}
