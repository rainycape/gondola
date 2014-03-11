// +build !appengine

package log

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

func isatty(w io.Writer) bool {
	if ioctlReadTermios != 0 {
		if f, ok := w.(*os.File); ok {
			var termios syscall.Termios
			_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(f.Fd()), ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
			return err == 0
		}
	}
	return false
}
