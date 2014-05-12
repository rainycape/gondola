// Package fixed implements a chunker which returns
// chunks of fixed size (except for the last one).
package fixed

import (
	"gnd.la/blobstore/chunk"
)

type chunker struct {
	buf    []byte
	pos    int
	writer chunk.Writer
}

func New(writer chunk.Writer, chunkSize int) chunk.Chunker {
	return &chunker{
		buf:    make([]byte, chunkSize),
		writer: writer,
	}
}

func (c *chunker) Write(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		nn := copy(c.buf[c.pos:], p[n:])
		c.pos += nn
		n += nn
		if c.pos == len(c.buf) {
			if err := c.Flush(); err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

func (c *chunker) Flush() error {
	var err error
	if c.pos > 0 {
		err = c.writer.WriteChunk(c.buf[:c.pos])
		c.Reset()
	}
	return err
}

func (c *chunker) Reset() {
	c.pos = 0
}

func (c *chunker) Remaining() []byte {
	return c.buf[:c.pos]
}
