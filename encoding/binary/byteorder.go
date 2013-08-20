package binary

// A ByteOrder specifies how to convert byte sequences into
// 16-, 32-, or 64-bit unsigned integers.
type ByteOrder struct {
	Uint16    func([]byte) uint16
	Uint32    func([]byte) uint32
	Uint64    func([]byte) uint64
	PutUint16 func([]byte, uint16)
	PutUint32 func([]byte, uint32)
	PutUint64 func([]byte, uint64)
	name      string
}

func (b *ByteOrder) String() string {
	return b.name
}

// LittleEndian is the little-endian implementation of ByteOrder.
var LittleEndian = &ByteOrder{
	Uint16:    leUint16,
	Uint32:    leUint32,
	Uint64:    leUint64,
	PutUint16: lePutUint16,
	PutUint32: lePutUint32,
	PutUint64: lePutUint64,
	name:      "LittleEndian",
}

// BigEndian is the big-endian implementation of ByteOrder.
var BigEndian = &ByteOrder{
	Uint16:    beUint16,
	Uint32:    beUint32,
	Uint64:    beUint64,
	PutUint16: bePutUint16,
	PutUint32: bePutUint32,
	PutUint64: bePutUint64,
	name:      "BigEndian",
}
