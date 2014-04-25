package driver

import (
	"strings"
)

// Fow now we only support funcs without arguments: now and today

func UnescapeDefault(val string) string {
	return strings.Replace(strings.Replace(val, "\\(", "(", -1), "\\)", ")", -1)
}

func IsFunc(val string) bool {
	return strings.HasSuffix(val, "()")
}

func SplitFuncArgs(val string) (string, []string) {
	if !IsFunc(val) {
		return "", nil
	}
	return strings.ToLower(strings.TrimSuffix(val, "()")), nil
}
