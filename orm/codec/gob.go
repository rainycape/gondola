package codec

import (
	"bytes"
	"encoding/gob"
	"reflect"
)

type gobCodec struct {
}

func (c *gobCodec) Name() string {
	return "gob"
}

func (c *gobCodec) Binary() bool {
	return true
}

func (c *gobCodec) Try(typ reflect.Type, tags []string) error {
	var buf bytes.Buffer
	val := reflect.New(typ)
	encoder := gob.NewEncoder(&buf)
	return encoder.Encode(val.Interface())
}

func (c *gobCodec) Encode(val *reflect.Value) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(val.Interface())
	return buf.Bytes(), err
}

func (c *gobCodec) Decode(data []byte, val *reflect.Value) error {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	return decoder.DecodeValue(*val)
}

func init() {
	Register(&gobCodec{})
}
