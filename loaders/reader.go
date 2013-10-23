package loaders

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

type readerAt interface {
	io.ReaderAt
	ReadSeekCloser
	Size() int64
}

// bytesReader just adds a fake Close method
// to bytes.Reader, so it satisfies the
// readerAt interface.
type bytesReader struct {
	*bytes.Reader
	size int64
}

func (r *bytesReader) Close() error {
	return nil
}

func (r *bytesReader) Size() int64 {
	return r.size
}

type stringReader struct {
	*strings.Reader
	size int64
}

func (r *stringReader) Close() error {
	return nil
}

func (r *stringReader) Size() int64 {
	return r.size
}

// fileReader just adds the Size() method
type fileReader struct {
	*os.File
}

func (r *fileReader) Size() int64 {
	if fi, err := r.Stat(); err == nil {
		return fi.Size()
	}
	return 0
}

type RawString string

func newReader(source interface{}) readerAt {
	switch s := source.(type) {
	case []byte:
		return &bytesReader{
			Reader: bytes.NewReader(s),
			size:   int64(len(s)),
		}
	case RawString:
		return &stringReader{
			Reader: strings.NewReader(string(s)),
			size:   int64(len(string(s))),
		}
	case string:
		f, err := os.Open(s)
		if err != nil {
			panic(err)
		}
		return &fileReader{f}
	case io.Reader:
		// TODO: Better implementation for io.Reader. Don't
		// read everything if the caller does not seek nor asks
		// for size.
		data, err := ioutil.ReadAll(s)
		if err != nil {
			panic(err)
		}
		if c, ok := s.(io.Closer); ok {
			c.Close()
		}
		return newReader(data)
	}
	panic(fmt.Errorf("invalid source type %T", source))
}
