package loaders

import (
	"io"
	"time"
)

type Loader interface {
	Load(name string) (ReadSeekCloser, time.Time, error)
	Create(name string) (io.WriteCloser, error)
}
