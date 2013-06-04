package driver

import (
	"strings"
)

type Tag string

func (t Tag) Name() string {
	pos := strings.Index(string(t), ",")
	if pos >= 0 {
		return string(t[:pos])
	}
	return string(t)
}

func (t Tag) Has(opt string) bool {
	s := string(t)
	return strings.Contains(s, ","+opt+",") || strings.Contains(s, ","+opt+":") || strings.HasSuffix(s, ","+opt)
}

func (t Tag) Value(key string) string {
	s := string(t)
	pos := strings.Index(s, key+":")
	if pos >= 0 {
		pos += len(key) + 1
		end := strings.Index(s[pos:], ",")
		if end < 0 {
			end = len(s)
		}
		return s[pos:end]
	}
	return ""
}
