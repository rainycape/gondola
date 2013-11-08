// Package serialize provides conveniency functions
// for serializing values to either JSON or XML
package serialize

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"runtime"
	"strconv"
)

type SerializationFormat int

const (
	Json SerializationFormat = iota
	Xml
)

// JSONWriter is the interface implemented by types which
// can write themselves as JSON into an io.Writer. You can
// use the gondola command for generating the code to implement
// this interface in your own types.
type JSONWriter interface {
	WriteJSON(w io.Writer) (int, error)
}

const bufSize = 4096

var (
	buffers = make(chan *bytes.Buffer, runtime.GOMAXPROCS(0))
)

func getBuffer() *bytes.Buffer {
	var buf *bytes.Buffer
	select {
	case buf = <-buffers:
		buf.Reset()
	default:
		buf = new(bytes.Buffer)
		buf.Grow(bufSize)
	}
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Len() <= bufSize {
		select {
		case buffers <- buf:
		default:
		}
	}
}

// Write serializes value using the SerializationFormat f and writes it
// to w. If w also implements http.ResponseWriter, it sets the appropiate
// headers too. Returns the number of bytes written and any error that might
// occur while serializing or writing the serialized data.
func Write(w io.Writer, value interface{}, f SerializationFormat) (int, error) {
	var data []byte
	var contentType string
	var err error
	switch f {
	case Json:
		switch v := value.(type) {
		case []byte:
			data = v
		case JSONWriter:
			buf := getBuffer()
			_, err = v.WriteJSON(buf)
			data = buf.Bytes()
			defer putBuffer(buf)
		default:
			// Unfortunately, there's no way to tell encoding/json
			// to use our own buffer. It will use its own and then
			// copy the result to our buffer, which is kinda
			// useless. Use value rather than v to avoid another
			// empty interface boxing.
			data, err = json.Marshal(value)
		}
		contentType = "application/json"
	case Xml:
		switch v := value.(type) {
		case []byte:
			data = v
		default:
			data, err = xml.Marshal(value)
		}
		contentType = "application/xml"
	default:
		panic("Invalid serialization format")
	}
	if err != nil {
		return 0, err
	}
	if rw, ok := w.(http.ResponseWriter); ok {
		header := rw.Header()
		header.Set("Content-Type", contentType)
		header.Set("Content-Length", strconv.Itoa(len(data)))
	}
	return w.Write(data)
}

// WriteJson is equivalent to Write(w, value, Json)
func WriteJson(w io.Writer, value interface{}) (int, error) {
	return Write(w, value, Json)
}

// WriteXml is equivalent to Write(w, value, Xml)
func WriteXml(w io.Writer, value interface{}) (int, error) {
	return Write(w, value, Xml)
}
