// +build appengine

package log

import (
	"io"

	"gnd.la/internal"
)

func isatty(w io.Writer) bool {
	if internal.InAppEngineDevServer() {
		return true
	}
	return false
}
