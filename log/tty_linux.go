// +build linux

package log

import (
	"syscall"
)

const ioctlReadTermios = syscall.TCGETS
