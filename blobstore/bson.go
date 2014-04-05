package blobstore

import (
	"gnd.la/internal/bson"
)

func marshal(in interface{}) ([]byte, error) {
	return bson.Marshal(in)
}

func unmarshal(data []byte, out interface{}) error {
	return bson.Unmarshal(data, out)
}

func newId() string {
	return bson.NewObjectId().Hex()
}
