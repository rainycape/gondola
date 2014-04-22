// +build !windows,!darwin

package runtimeutil

import (
	"debug/elf"
	"fmt"
	"io"
)

type file struct {
	*elf.File
}

func (f *file) section(name string) ([]byte, error) {
	s := f.Section(name)
	if s == nil {
		return nil, fmt.Errorf("no section name %q", name)
	}
	return s.Data()
}

func (f *file) Symtab() ([]byte, error) {
	return f.section(".gosymtab")
}

func (f *file) Pclntab() ([]byte, error) {
	return f.section(".gopclntab")
}

func (f *file) TextAddr() uint64 {
	return f.Section(".text").Addr
}

func openDebugFile(r io.ReaderAt) (debugFile, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	return &file{f}, nil
}
