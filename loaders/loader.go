package loaders

import (
	"io"
	"time"
)

type Loader interface {
	Load(name string) (ReadSeekCloser, time.Time, error)
	Create(name string, overwrite bool) (io.WriteCloser, error)
}

type Lister interface {
	List() ([]string, error)
}
