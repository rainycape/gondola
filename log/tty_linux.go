// +build linux,!appengine

package log

import (
	"syscall"
)

const ioctlReadTermios = syscall.TCGETS
