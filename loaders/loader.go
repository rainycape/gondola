package loaders

import (
	"io"
	"time"
)

type Loader interface {
	Load(name string) (ReadSeekCloser, time.Time, error)
	List() ([]string, error)
	Create(name string, overwrite bool) (io.WriteCloser, error)
}
