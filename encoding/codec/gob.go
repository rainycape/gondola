package codec

import (
	"bytes"
	"encoding/gob"
)

var (
	gobCodec = &Codec{Encode: gobMarshal, Decode: gobUnmarshal, Binary: true}
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
	Register("gob", gobCodec)
}
