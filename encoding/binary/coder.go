package binary

type coder struct {
	order *ByteOrder
	buf   [8]byte
	err   error
}
