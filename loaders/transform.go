package loaders

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"
)

// transformer is called to transform the data
// before passing it to the final loader when writing
// and after obtaining the data from the loader when
// reading.
type transformer func([]byte) ([]byte, error)

type twriter struct {
	buf         bytes.Buffer
	transformer transformer
	writer      io.WriteCloser
}

func (t *twriter) Write(b []byte) (int, error) {
	return t.buf.Write(b)
}

func (t *twriter) Close() error {
	d, err := t.transformer(t.buf.Bytes())
	if err != nil {
		return err
	}
	if _, err := t.writer.Write(d); err != nil {
		return err
	}
	return t.writer.Close()
}

// transformLoader
type transformLoader struct {
	readTransformer  transformer
	writeTransformer transformer
	loader           Loader
}

func (t *transformLoader) Load(name string) (ReadSeekCloser, time.Time, error) {
	r, tm, err := t.loader.Load(name)
	if t.readTransformer != nil && err == nil {
		var b []byte
		b, err = ioutil.ReadAll(r)
		if err == nil {
			b, err = t.readTransformer(b)
			if err == nil {
				return newReader(b), tm, nil
			}
		}
	}
	return r, tm, err
}

func (t *transformLoader) Create(name string) (io.WriteCloser, error) {
	w, err := t.loader.Create(name)
	if t.writeTransformer != nil && err == nil {
		return &twriter{
			transformer: t.writeTransformer,
			writer:      w,
		}, nil
	}
	return w, err
}
