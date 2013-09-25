package paginator

import (
	"fmt"
	"strings"
)

type Func func(base string, page int) string

func MakeFmtFunc(format string) Func {
	return func(base string, page int) string {
		if page == 1 {
			return base
		}
		return strings.Replace(fmt.Sprintf(format, base, page), "//", "/", -1)
	}
}
