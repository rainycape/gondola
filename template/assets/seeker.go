package assets

import (
	"bytes"
	"io"
	"io/ioutil"
)

type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

type closer struct {
	io.ReadSeeker
}

func (c *closer) Close() error {
	if cl, ok := c.ReadSeeker.(io.Closer); ok {
		return cl.Close()
	}
	return nil
}

func Seeker(r io.Reader) (ReadSeekCloser, error) {
	if rsc, ok := r.(ReadSeekCloser); ok {
		return rsc, nil
	}
	if rs, ok := r.(io.ReadSeeker); ok {
		return &closer{rs}, nil
	}
	// No seeking support, read all the data into a buffer
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &closer{bytes.NewReader(data)}, nil
}
