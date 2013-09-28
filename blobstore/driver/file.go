package driver

import (
	"io"
)

type WFile interface {
	io.WriteCloser
	io.Seeker
}

type RFile interface {
	io.ReadSeeker
	io.Closer
}
