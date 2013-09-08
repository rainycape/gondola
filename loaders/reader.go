package loaders

import (
	"bytes"
)

// reader just adds a fake Close method
// to bytes.Reader, so it satisfies the
// ReadSeekCloser interface.
type reader struct {
	*bytes.Reader
}

func (r *reader) Close() error {
	return nil
}

func newReader(b []byte) *reader {
	return &reader{
		Reader: bytes.NewReader(b),
	}
}
