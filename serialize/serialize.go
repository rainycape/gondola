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

func Write(w io.Writer, value interface{}, f SerializationFormat) (int, error) {
	var contentType string
	var data []byte
	var err error
	switch f {
	case Json:
		data, err = json.Marshal(value)
		contentType = "application/json"
	case Xml:
		data, err = xml.Marshal(value)
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

func WriteJson(w io.Writer, value interface{}) (int, error) {
	return Write(w, value, Json)
}

func WriteXml(w io.Writer, value interface{}) (int, error) {
	return Write(w, value, Xml)
}
