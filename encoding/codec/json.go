package codec

import (
	"encoding/json"
)

var (
	jsonCodec = &Codec{Encode: json.Marshal, Decode: json.Unmarshal}
)

func init() {
	Register("json", jsonCodec)
}
