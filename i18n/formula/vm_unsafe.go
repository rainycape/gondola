// +build !appengine

package formula

import (
	"unsafe"
)

func bint(b bool) int {
	p := (*uint8)(unsafe.Pointer(&b))
	return int(*p)
}
