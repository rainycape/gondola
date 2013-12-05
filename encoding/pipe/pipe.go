// Package pipe implements pipes which transform data,
// generally for compressing it.
//
// This package includes the pipe "zlib", which compreses
// any value bigger than 100 bytes using the default zlib
// compression level. To tell compressed and uncompressed
// data apart, it prepends a byte to its output (0 for
// uncompressed, 1 for compressed).
package pipe

import (
	"fmt"
	"gnd.la/util/structs"
)

var (
	registry = map[string]*Pipe{}
)

// Pipe represents a codec pipe, which can encode and decode data
// as it's saved to or loaded from the database.
type Pipe struct {
	// Encode passes the given data trough the pipe and produces
	// a new ouput. len(data) is always > 0.
	Encode func(data []byte) ([]byte, error)
	// Decode performs the inverse operation of Encode, producing
	// the original input to Encode from its output.
	// len(data) is always > 0.
	Decode func(data []byte) ([]byte, error)
}

// Register registers a new Pipe into the pipe registry.
// If there's already a Pipe with the same name, it will panic.
func Register(name string, p *Pipe) {
	if _, ok := registry[name]; ok {
		panic(fmt.Errorf("there's already a codec pipe named %q", name))
	}
	registry[name] = p
}

// Get returns the Pipe with the give name, or nil if there's no such Pipe.
func Get(name string) *Pipe {
	return registry[name]
}

// FromTag returns the pipe for a given field tag.
func FromTag(t *structs.Tag) *Pipe {
	return registry[t.PipeName()]
}
