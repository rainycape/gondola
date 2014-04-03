package runtimeutil

import (
	"debug/elf"
	"io"
)

type file struct {
	*elf.File
}

func (f *file) Addr(name string) uint64 {
	return f.File.Section(name).Addr
}

func (f *file) Section(name string) section {
	return f.File.Section(name)
}

func openDebugFile(r io.ReaderAt) (debugFile, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	return &file{f}, nil
}
