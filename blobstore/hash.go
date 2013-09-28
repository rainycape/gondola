package blobstore

import (
	"hash"
	"hash/fnv"
)

func newHash() hash.Hash64 {
	return fnv.New64a()
}
