package runtimeutil

type section interface {
	Data() ([]byte, error)
}

type debugFile interface {
	Section(name string) section
	Addr(name string) uint64
	Close() error
}
