// +build !go1.3

package template

import (
	"bytes"
	"runtime"
)

var (
	poolSize  = runtime.GOMAXPROCS(0)
	statePool = make(chan *state, poolSize)
	bufPool   = make(chan *bytes.Buffer, poolSize)
)

func getState() *state {
	select {
	case s := <-statePool:
		return s
	default:
	}
	return nil
}

func putState(s *state) {
	select {
	case statePool <- s:
	default:
	}
}

func getBuffer() *bytes.Buffer {
	var buf *bytes.Buffer
	select {
	case buf = <-bufPool:
		buf.Reset()
	default:
		buf = new(bytes.Buffer)
	}
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	select {
	case bufPool <- buf:
	default:
	}
}
