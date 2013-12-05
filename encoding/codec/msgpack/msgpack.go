package msgpack

import (
	gocodec "github.com/ugorji/go/codec"
	"gnd.la/encoding/codec"
)

var (
	msgpackCodec = &codec.Codec{Encode: msgpackMarshal, Decode: msgpackUnmarshal, Binary: true}
	handle       = &gocodec.MsgpackHandle{}
)

func msgpackMarshal(in interface{}) ([]byte, error) {
	var b []byte
	enc := gocodec.NewEncoderBytes(&b, handle)
	err := enc.Encode(in)
	return b, err
}

func msgpackUnmarshal(data []byte, out interface{}) error {
	dec := gocodec.NewDecoderBytes(data, handle)
	return dec.Decode(out)
}

func init() {
	codec.Register("msgpack", msgpackCodec)
}
