package paginator

import "fmt"

// Pager represents an interface which returns the URL
// for the given page number. Note that page numbers are
// 1-indexed (i.e. the first page is page 1, not page 0).
type Pager interface {
	URL(page int) string
}

type pager func(int) string

func (p pager) URL(page int) string {
	return p(page)
}

// Fmt returns a Pager which returns the base URL for
// the first page and then fmt.Sprintf(format, base, page) for
// other pages.
func Fmt(format string, base string) Pager {
	return pager(func(page int) string {
		if page == 1 {
			return base
		}
		return fmt.Sprintf(format, base, page)
	})
}
