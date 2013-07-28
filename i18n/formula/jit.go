package formula

import (
	"errors"
)

var (
	errJitNotSupported = errors.New("formula JIT is not supported on this os/arch")
	errEmptyProgram    = errors.New("can't JIT an empty program")
)
