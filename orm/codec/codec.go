package codec

import (
	"fmt"
	"gnd.la/types"
	"reflect"
)

var (
	registry = map[string]Codec{}
)

// Codec defines the interface to be implemented by ORM codecs.
type Codec interface {
	// Name returns the codec name, which will be used in the field tags.
	// e.g. given a codec named "json" the field should be
	// tagged like:
	//
	//     type Foo struct {
	//     ...
	//         Bars []Bar `orm:",codec=json"`
	//     }
	Name() string
	// Binary returns wheter the codec returns binary or text data.
	Binary() bool
	// Try tries to encode a given type, returns an error if
	// the type can't be encoded by the codec.
	Try(typ reflect.Type, tags []string) error
	// Encode encodes the given value and returns its encoded
	// representation as []byte and any error that might
	// happen while encoding it.
	Encode(val *reflect.Value) ([]byte, error)
	// Decode decodes the given data into the given value and
	// returns any errors that might happen while decoding.
	// The value is guaranteed to be addressable.
	Decode(data []byte, val *reflect.Value) error
}

// Register registers a new Codec into the codec registry.
// If there's already a codec with the same name, it will panic.
func Register(c Codec) {
	name := c.Name()
	if _, ok := registry[name]; ok {
		panic(fmt.Errorf("there's already an ORM codec named %q", name))
	}
	registry[name] = c
}

// Get returns the codec with the give name, or nil if there's no such codec.
func Get(name string) Codec {
	return registry[name]
}

// FromTag returns the codec for a given field tag.
func FromTag(t *types.Tag) Codec {
	return registry[t.CodecName()]
}
