// +build !appengine,amd64

package binary

//go:noescape

func (littleEndian) Uint16(b []byte) uint16

//go:noescape

func (littleEndian) PutUint16(b []byte, v uint16)

//go:noescape

func (littleEndian) Uint32(b []byte) uint32

//go:noescape

func (littleEndian) PutUint32(b []byte, v uint32)

//go:noescape

func (littleEndian) Uint64(b []byte) uint64

//go:noescape

func (littleEndian) PutUint64(b []byte, v uint64)

//go:noescape

func (bigEndian) Uint16(b []byte) uint16

//go:noescape

func (bigEndian) PutUint16(b []byte, v uint16)

//go:noescape

func (bigEndian) Uint32(b []byte) uint32

//go:noescape

func (bigEndian) PutUint32(b []byte, v uint32)

//go:noescape

func (bigEndian) Uint64(b []byte) uint64

//go:noescape

func (bigEndian) PutUint64(b []byte, v uint64)
