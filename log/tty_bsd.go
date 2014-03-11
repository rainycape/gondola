// +build darwin

package log

import (
	"syscall"
)

const ioctlReadTermios = syscall.TIOCGETA
