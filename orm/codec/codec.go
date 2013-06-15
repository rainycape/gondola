package codec

import (
	"fmt"
	"gondola/orm/tag"
	"reflect"
)

var (
	registry = map[string]Codec{}
)

type Codec interface {
	Name() string
	Binary() bool
	Try(typ reflect.Type, tags []string) error
	Encode(val *reflect.Value) ([]byte, error)
	Decode(data []byte, val *reflect.Value) error
}

func Register(c Codec) {
	name := c.Name()
	if _, ok := registry[name]; ok {
		panic(fmt.Errorf("there's already an ORM codec named %q", name))
	}
	registry[name] = c
}

func Get(name string) Codec {
	return registry[name]
}

func FromTag(t *tag.Tag) Codec {
	return registry[t.CodecName()]
}
