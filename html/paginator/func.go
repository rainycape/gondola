package paginator

import (
	"fmt"
	"strings"
)

// Func receives the base URL (which might be relative or
// absolute) and the page number and returns the URL for
// the requested page number, as a string.
type Func func(base string, page int) string

// Fmt returns a function which returns the base URL for
// the first page and then fmt.Sprintf(format, base, page) for
// other pages.
func Fmt(format string) Func {
	return func(base string, page int) string {
		if page == 1 {
			return base
		}
		// TODO: This fails for absolute URLs, breaks the ://
		return strings.Replace(fmt.Sprintf(format, base, page), "//", "/", -1)
	}
}
