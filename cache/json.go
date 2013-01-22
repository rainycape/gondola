package cache

import (
	"bytes"
	"encoding/json"
)

var (
	JsonEncoder = Codec{Encode: jsonMarshal, Decode: jsonUnmarshal}
)

func jsonMarshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

func init() {
	RegisterCodec("json", &JsonEncoder)
}
