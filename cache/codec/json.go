package codec

import (
	"encoding/json"
)

var (
	JsonCodec = &Codec{Encode: json.Marshal, Decode: json.Unmarshal}
)

func init() {
	Register("json", JsonCodec)
}
