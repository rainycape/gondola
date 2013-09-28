package driver

import (
	"io"
)

type WFile interface {
	io.WriteCloser
}

type RFile interface {
	io.ReadSeeker
	io.Closer
}
