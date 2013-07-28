// +build !appengine,linux

package formula

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

var (
	lock sync.Mutex
	page []byte
	used int
)

func makeJitFunc(code []byte) (Formula, error) {
	lock.Lock()
	defer lock.Unlock()
	if unused := len(page) - used; unused < len(code) {
		m, err := syscall.Mmap(0, 0, os.Getpagesize(), 0, syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS)
		if err != nil {
			return nil, err
		}
		page = m
		used = 0
	}
	if err := syscall.Mprotect(page, syscall.PROT_WRITE|syscall.PROT_EXEC); err != nil {
		return nil, err
	}
	p := page[used:]
	copy(p, code)
	if err := syscall.Mprotect(page, syscall.PROT_EXEC); err != nil {
		return nil, err
	}
	used += len(code)
	start := &p
	entry := (**[]byte)(unsafe.Pointer(&start))
	return *(*Formula)(unsafe.Pointer(entry)), nil
}
