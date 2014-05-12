package chunk

import (
	"io"
)

// Chunker is the interface implemented by all the chunkers.
// A Chunker receives a stream of data and splits it into
// chunks, which are written by its ChunkWriter (usually
// implemented by the caller.
type Chunker interface {
	io.Writer
	// Flush causes the chunker to write its current buffer
	// using its ChunkWriter.
	Flush() error
	// Reset prepares the chunker for a new chunk of data.
	// Chunkers should automatically reset themselves after
	// being flushed.
	Reset()
	// Remaining returns the data in the chunker's buffer.
	Remaining() []byte
}

// Writer defines the interface which a Chunker uses to write the
// chunked data.
type Writer interface {
	// WriteChunk will be called once per chunk and all chunks
	// will be non-empty.
	WriteChunk(b []byte) error
}
