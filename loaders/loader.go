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
}
