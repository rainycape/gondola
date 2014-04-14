// +build darwin,!appengine

package log

import (
	"syscall"
)

const ioctlReadTermios = syscall.TIOCGETA
