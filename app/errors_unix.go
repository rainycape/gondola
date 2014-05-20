// +build !windows,!appengine

package app

import (
	"syscall"
)

var (
	ePIPE      = syscall.EPIPE
	eCONNRESET = syscall.ECONNRESET
)
