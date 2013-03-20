package serialize

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strconv"
)

type Serializer int

const (
	Json Serializer = iota
	Xml
)

func Write(w http.ResponseWriter, value interface{}, s Serializer) (int, error) {
	var contentType string
	var data []byte
	var err error
	if s == Json {
		data, err = json.Marshal(value)
		contentType = "application/json"
	} else if s == Xml {
		data, err = xml.Marshal(value)
		contentType = "application/xml"
	}
	if err != nil {
		return 0, err
	}
	total := len(data)
	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Content-Length", strconv.Itoa(total))
	for c := 0; c < total; {
		n, err := w.Write(data)
		c += n
		if err != nil {
			return c, err
		}
	}
	return total, nil
}
