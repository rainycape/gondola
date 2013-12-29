// +build !appengine

package formula

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

type mmapLinux struct {
	lock sync.Mutex
	page []byte
	used int
}

type Formula32 func(int32) int32

func (m *mmapLinux) Map(code []byte) (Formula32, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if unused := len(m.page) - m.used; unused < len(code) {
		page, err := syscall.Mmap(0, 0, os.Getpagesize(), 0, syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS)
		if err != nil {
			return nil, err
		}
		m.page = page
		m.used = 0
	}
	if err := syscall.Mprotect(m.page, syscall.PROT_WRITE|syscall.PROT_EXEC); err != nil {
		return nil, err
	}
	p := m.page[m.used:]
	copy(p, code)
	if err := syscall.Mprotect(m.page, syscall.PROT_EXEC); err != nil {
		return nil, err
	}
	m.used += len(code)
	start := &p
	entry := (**[]byte)(unsafe.Pointer(&start))
	return *(*Formula32)(unsafe.Pointer(entry)), nil
}

func init() {
	mmap = &mmapLinux{}
}
