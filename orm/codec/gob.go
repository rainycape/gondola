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
	switch typ.Kind() {
	case reflect.Slice:
		return c.Try(typ.Elem(), tags)
	case reflect.Map:
		if err := c.Try(typ.Key(), tags); err != nil {
			return err
		}
		return c.Try(typ.Elem(), tags)
	default:
		val := reflect.New(typ)
		b, err := c.Encode(&val)
		if err != nil {
			return err
		}
		return c.Decode(b, &val)
	}
}

func (c *gobCodec) Encode(val *reflect.Value) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.EncodeValue(*val)
	return buf.Bytes(), err
}

func (c *gobCodec) Decode(data []byte, val *reflect.Value) error {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	return decoder.DecodeValue(*val)
}

func init() {
	Register(&gobCodec{})
}
