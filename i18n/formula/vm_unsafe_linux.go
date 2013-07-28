// +build !appengine

package formula

import (
	"syscall"
	"unsafe"
)

func makeJitFunc(code []byte) (Formula, error) {
	m, err := syscall.Mmap(0, 0, len(code), syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC, syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS)
	if err != nil {
		return nil, err
	}
	copy(m, code)
	start := &m
	entry := (**[]byte)(unsafe.Pointer(&start))
	return *(*Formula)(unsafe.Pointer(entry)), nil
}
