package loaders

import (
	"archive/zip"
	"time"
)

type zipLoader struct {
	mapLoader
	reader *zip.Reader
	err    error
}

func (z *zipLoader) List() ([]string, error) {
	if z.err != nil {
		return nil, z.err
	}
	names, err := z.mapLoader.List()
	if err != nil {
		return nil, err
	}
	for _, v := range z.reader.File {
		names = append(names, v.Name)
	}
	return names, nil
}

func (z *zipLoader) Load(name string) (ReadSeekCloser, time.Time, error) {
	if z.err != nil {
		return nil, time.Time{}, z.err
	}
	for _, v := range z.reader.File {
		if v.Name == name {
			rc, err := v.Open()
			if err != nil {
				return nil, time.Time{}, err
			}
			return newReader(rc), v.ModTime(), nil
		}
	}
	return z.mapLoader.Load(name)
}

// ZipLoader returns a Loader which loads resources from the
// given zip file. souce must be either a filename, a []byte,
// a RawString or a io.Reader.
func ZipLoader(source interface{}) Loader {
	r := newReader(source)
	reader, err := zip.NewReader(r, r.Size())
	return &zipLoader{
		mapLoader: mapLoader{},
		reader:    reader,
		err:       err,
	}
}
