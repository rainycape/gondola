// +build windows, !appengine

package log

import (
	"io"
)

func isatty(w io.Writer) bool {
	return false
}
