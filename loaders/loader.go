package loaders

import (
	"io"
	"time"
)

type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

type Loader interface {
	Load(name string) (ReadSeekCloser, time.Time, error)
	Create(name string) (io.WriteCloser, error)
}
