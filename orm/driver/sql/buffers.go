package sql

import (
	"bytes"
	"sync"
)

var (
	bufferPool sync.Pool
)

func getBuffer() *bytes.Buffer {
	if x := bufferPool.Get(); x != nil {
		buf := x.(*bytes.Buffer)
		buf.Reset()
		return buf
	}
	return new(bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	bufferPool.Put(buf)
}
