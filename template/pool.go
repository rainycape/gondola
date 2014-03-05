// +build go1.3

package template

import (
	"bytes"
	"sync"
)

var (
	statePool sync.Pool
	bufPool   sync.Pool
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
