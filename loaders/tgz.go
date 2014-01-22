package loaders

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"time"
)

type tgzLoader struct {
	mapLoader
	reader *gzip.Reader
	err    error
}

func (t *tgzLoader) decode() error {
	if t.err != nil {
		return t.err
	}
	if t.reader != nil {
		defer t.reader.Close()
		tr := tar.NewReader(t.reader)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			w, err := t.Create(hdr.Name, false)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, tr); err != nil {
				return err
			}
			if err := w.Close(); err != nil {
				return err
			}
		}
		t.reader = nil
	}
	return nil
}

func (t *tgzLoader) List() ([]string, error) {
	if err := t.decode(); err != nil {
		return nil, err
	}
	return t.mapLoader.List()
}

func (t *tgzLoader) Load(name string) (ReadSeekCloser, time.Time, error) {
	if err := t.decode(); err != nil {
		return nil, time.Time{}, err
	}
	return t.mapLoader.Load(name)
}

// ThzLoader returns a Loader which loads resources from the
// given tgz file. souce must be either a filename, a []byte,
// a RawString or a io.Reader.
func TgzLoader(source interface{}) Loader {
	r := newReader(source)
	reader, err := gzip.NewReader(r)
	return &tgzLoader{
		mapLoader: mapLoader{},
		reader:    reader,
		err:       err,
	}
}
