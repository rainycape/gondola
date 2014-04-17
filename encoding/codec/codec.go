// Package codec implements functions for encoding and decoding
// objects in several formats
//
// Any registered codec can be used by both gnd.la/cache and
// gnd.la/orm.
//
// This package provides the "gob" and "json" codecs, which encode
// the data using encoding/gob and encoding/json, respectivelly.
// Check gnd.la/cache and gnd.la/orm to learn how to use codecs with
// Gondola's cache and ORM.
//
// Users might define their own codecs by implementing a Codec
// struct and registering it with Register().
package codec

import (
	"gnd.la/util/structs"
)

var (
	codecs  = map[string]*Codec{}
	imports = map[string]string{
		"msgpack": "gnd.la/encoding/codec/msgpack",
	}
)

// Codec is a stateless struct which implements
// encoding and decoding objects into/from []byte.
// An struct rather than an interface is used to
// enforce them to be stateless
// and also for performance reasons.
type Codec struct {
	// Encode takes an object and returns its []byte representation.
	Encode func(v interface{}) ([]byte, error)
	// Decode takes a []byte representation and decodes it into
	// the passed in object.
	Decode func(data []byte, v interface{}) error
	// Binary indicates if the codec returns binary or text data
	Binary bool
}

// Register registers a codec to be made available for
// use by the cache. If there was already a codec with the
// same name, it's overwritten by the new one. Keep in mind
// that this function is not thread safe, so it should only
// be called from the main goroutine.
func Register(name string, c *Codec) {
	codecs[name] = c
}

// Get returns the codec assocciated with the given name, or
// nil if there's no such codec.
func Get(name string) *Codec {
	return codecs[name]
}

// FromTag returns the pipe for a given field tag.
func FromTag(t *structs.Tag) *Codec {
	return codecs[t.CodecName()]
}

// RequiredImport returns the import required
// for using the codec with the given name, or
// the empty string if the codec is not known.
func RequiredImport(name string) string {
	return imports[name]
}
