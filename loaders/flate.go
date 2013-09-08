package loaders

import (
	"bytes"
	"compress/flate"
	"io/ioutil"
)

func flateDecompress(b []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(b))
	defer r.Close()
	return ioutil.ReadAll(r)
}

func flateCompress(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func FlateLoader(loader Loader) Loader {
	return &transformLoader{
		readTransformer:  flateDecompress,
		writeTransformer: flateCompress,
		loader:           loader,
	}
}
