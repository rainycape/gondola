package template

import (
	"bytes"

	"gopkgs.com/pool.v1"
)

var (
	statePool = pool.New(0)
	bufPool   = pool.New(0)
)

func getState() *State {
	s, _ := statePool.Get().(*State)
	return s
}

func putState(s *State) {
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
