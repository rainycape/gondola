// +build !appengine

package log

// #include <unistd.h>
import "C"
import (
	"io"
	"os"
)

func isatty(w io.Writer) bool {
	is := false
	if f, ok := w.(*os.File); ok {
		if C.isatty(C.int(f.Fd())) > 0 {
			is = true
		}
	}
	return is
}
