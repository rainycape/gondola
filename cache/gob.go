package cache

import (
	"bytes"
	"encoding/gob"
)

var (
	GobEncoder = Codec{Encode: gobMarshal, Decode: gobUnmarshal}
)

func gobMarshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gobUnmarshal(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

func init() {
	RegisterCodec("gob", &GobEncoder)
}
