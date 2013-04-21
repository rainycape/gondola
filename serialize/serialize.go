// Package serialize provides conveniency functions
// for serializing values to either JSON or XML
package serialize

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
)

type SerializationFormat int

const (
	Json SerializationFormat = iota
	Xml
)

// Write serializes value using the SerializationFormat f and writes it
// to w. If w also implements http.ResponseWriter, it sets the appropiate
// headers too. Returns the number of bytes written and any error that might
// occur while serializing or writing the serialized data.
func Write(w io.Writer, value interface{}, f SerializationFormat) (int, error) {
	data, _ := value.([]byte)
	var contentType string
	var err error
	switch f {
	case Json:
		if data == nil {
			data, err = json.Marshal(value)
		}
		contentType = "application/json"
	case Xml:
		if data == nil {
			data, err = xml.Marshal(value)
		}
		contentType = "application/xml"
	default:
		panic("Invalid serialization format")
	}
	if err != nil {
		return 0, err
	}
	total := len(data)
	if rw, ok := w.(http.ResponseWriter); ok {
		header := rw.Header()
		header.Set("Content-Type", contentType)
		header.Set("Content-Length", strconv.Itoa(total))
	}
	for c := 0; c < total; {
		n, err := w.Write(data)
		c += n
		if err != nil {
			return c, err
		}
	}
	return total, nil
}

// WriteJson is equivalent to Write(w, value, Json)
func WriteJson(w io.Writer, value interface{}) (int, error) {
	return Write(w, value, Json)
}

// WriteXml is equivalent to Write(w, value, Xml)
func WriteXml(w io.Writer, value interface{}) (int, error) {
	return Write(w, value, Xml)
}
