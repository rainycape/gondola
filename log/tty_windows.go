// +build windows

package log

import (
	"io"
)

func isatty(w io.Writer) bool {
	return false
}
