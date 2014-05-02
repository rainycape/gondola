// +build !appengine

package sql

import (
	"unsafe"
)

func stobs(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

func bstos(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
