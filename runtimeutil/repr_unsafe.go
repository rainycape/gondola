// +build !appengine

package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"reflect"
	"unsafe"
)

type emptyInterface struct {
	typ uintptr
	val uintptr
}

func stringRepr(val uint64) string {
	s := (*string)(unsafe.Pointer(uintptr(val)))
	return fmt.Sprintf("= %q", *s)
}

func emptyInterfaceRepr(val1 uint64, val2 uint64) string {
	if val1 == 0 || val2 == 0 {
		return pointerRepr(val2)
	}
	i := *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: uintptr(val1),
		val: uintptr(val2),
	}))
	return fmt.Sprintf("= %T(%v)", i, i)
}

func typeName(table *gosym.Table, fn *gosym.Func, s *gosym.Sym) string {
	i := *(*interface{})(unsafe.Pointer(&s.GoType))
	return reflect.TypeOf(i).String()
}

func isInterface(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, tn string) bool {
	i := *(*interface{})(unsafe.Pointer(&s.GoType))
	return reflect.TypeOf(i).Kind() == reflect.Interface
}
