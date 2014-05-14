package driver

import (
	"errors"
	"io"
)

var (
	ErrMetadataNotHandled = errors.New("this driver does not handle metadata")
)

type WFile interface {
	io.WriteCloser
	SetMetadata([]byte) error
}

type RFile interface {
	io.ReadSeeker
	io.Closer
	Metadata() ([]byte, error)
}
