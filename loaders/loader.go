package loaders

import (
	"io"
	"time"
)

type ReadSeekerCloser interface {
	io.ReadCloser
	io.Seeker
}

type Loader interface {
	Load(name string) (ReadSeekerCloser, time.Time, error)
	MkTemp(prefix, ext string) (io.WriteCloser, string, error)
	Create(name string) (io.WriteCloser, error)
}
