// Package codec implements functions for encoding and decoding
// objects which are stored in Gondola's cache.
//
// This package provides the "gob" and "json" codecs, which encode
// the data using encoding/gob and encoding/json, respectivelly.
// Check the cache package documentation to learn how to choose
// a codec when initialiazing a cache instance.
//
// Users might define their own codecs by implementing a Codec
// struct and registering it with Register().
package codec

var (
	codecs = map[string]*Codec{}
)

// Codec is a stateless struct which implements
// encoding and decoding objects into/from []byte
// for storing them in the cache. An struct rather than
// an interface is used to enforce them to be stateless
// and also for performance reasons.
type Codec struct {
	// Encode takes an object and returns its []byte representation.
	Encode func(v interface{}) ([]byte, error)
	// Decode takes a []byte representation and decodes it into
	// the passed in object.
	Decode func(data []byte, v interface{}) error
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
