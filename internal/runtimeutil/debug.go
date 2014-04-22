package runtimeutil

type debugFile interface {
	Symtab() ([]byte, error)
	Pclntab() ([]byte, error)
	TextAddr() uint64
	Close() error
}
