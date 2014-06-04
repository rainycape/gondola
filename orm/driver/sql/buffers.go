package sql

import (
	"bytes"

	"gopkgs.com/pool.v1"
)

var (
	bufferPool = pool.New(0)
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
