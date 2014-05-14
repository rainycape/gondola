package profile

import (
	"io"
)

type reader struct {
	r    io.Reader
	name string
	note string
}

func (r *reader) Read(p []byte) (int, error) {
	defer Start(r.name).Note("READ", r.note).End()
	return r.r.Read(p)
}

type readCloser struct {
	reader
	r io.ReadCloser
}

func (r *readCloser) Close() error {
	return r.r.Close()
}

func newReader(r io.Reader, name string, note string) *reader {
	return &reader{
		r:    r,
		name: name,
		note: note,
	}
}

// Reader returns an io.Reader which creates a timed profiling
// event everytime it's read from.
func Reader(r io.Reader, name string, note string) io.Reader {
	return newReader(r, name, note)
}

// ReadCloser works like Reader, but wraps an io.ReadCloser.
func ReadCloser(r io.ReadCloser, name string, note string) io.ReadCloser {
	reader := newReader(r, name, note)
	return &readCloser{
		reader: *reader,
		r:      r,
	}
}
