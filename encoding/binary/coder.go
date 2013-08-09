package binary

import (
	"runtime"
)

type coder struct {
	order ByteOrder
	buf   [8]byte
	err   error
}

func (c *coder) recover() {
	if r := recover(); r != nil {
		if _, ok := r.(runtime.Error); ok {
			panic(r)
		}
		c.err = r.(error)
	}
}
