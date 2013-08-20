// +build !appengine,amd64

package binary

//go:noescape

func leUint16(b []byte) uint16

//go:noescape

func lePutUint16(b []byte, v uint16)

//go:noescape

func leUint32(b []byte) uint32

//go:noescape

func lePutUint32(b []byte, v uint32)

//go:noescape

func leUint64(b []byte) uint64

//go:noescape

func lePutUint64(b []byte, v uint64)

//go:noescape

func beUint16(b []byte) uint16

//go:noescape

func bePutUint16(b []byte, v uint16)

//go:noescape

func beUint32(b []byte) uint32

//go:noescape

func bePutUint32(b []byte, v uint32)

//go:noescape

func beUint64(b []byte) uint64

//go:noescape

func bePutUint64(b []byte, v uint64)
