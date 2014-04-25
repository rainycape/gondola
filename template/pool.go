package template

import (
	"bytes"

	"gnd.la/internal/pool"
)

var (
	statePool = pool.New(0)
	bufPool   = pool.New(0)
)

func getState() *state {
	s, _ := statePool.Get().(*state)
	return s
}

func putState(s *state) {
	statePool.Put(s)
}

func getBuffer() *bytes.Buffer {
	v := bufPool.Get()
	if v != nil {
		b := v.(*bytes.Buffer)
		b.Reset()
		return b
	}
	return new(bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	bufPool.Put(buf)
}

func newBuffer() interface{} {
	return new(bytes.Buffer)
}
