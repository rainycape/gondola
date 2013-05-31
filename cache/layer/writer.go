package layer

import (
	"bytes"
	"net/http"
)

type writer struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
	header     http.Header
}

func (w *writer) copyHeaders() {
	if w.header == nil {
		w.header = http.Header{}
		for k, v := range w.ResponseWriter.Header() {
			w.header[k] = v
		}
		if w.statusCode == 0 {
			w.statusCode = http.StatusOK
		}
	}
}

func (w *writer) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *writer) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if err == nil && n > 0 {
		w.buf.Write(data)
		if w.header == nil {
			w.copyHeaders()
		}
	}
	return n, err
}

func newWriter(rw http.ResponseWriter) *writer {
	return &writer{
		ResponseWriter: rw,
		buf:            bytes.NewBuffer(nil),
	}
}
