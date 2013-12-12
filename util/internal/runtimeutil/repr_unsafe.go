// +build !appengine

package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"gnd.la/html"
	"reflect"
	"strconv"
	"unsafe"
)

type emptyInterface struct {
	typ uintptr
	val uintptr
}

func pointerRepr(val uint64, s *gosym.Sym, _html bool) string {
	if val == 0 {
		return "= nil"
	}
	p := strconv.FormatUint(val, 16)
	if _html && s != nil {
		t := reflectType(s.GoType)
		if t.Kind() == reflect.Map || (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct) {
			val := reflect.NewAt(t, unsafe.Pointer(&val))
			title := fmt.Sprintf("%+v", val.Elem().Interface())
			return fmt.Sprintf("@ <abbr title=\"%s\">0x%s</abbr>", html.Escape(title), p)
		}
	}
	return "@ 0x" + p
}

func reflectType(goType uint64) reflect.Type {
	i := *(*interface{})(unsafe.Pointer(&goType))
	return reflect.TypeOf(i)
}

func stringRepr(val1 uint64, val2 uint64) string {
	sh := &reflect.StringHeader{
		Data: uintptr(val1),
		Len:  int(val2),
	}
	s := (*string)(unsafe.Pointer(sh))
	return fmt.Sprintf("= %q", *s)
}

func sliceRepr(val1 uint64, val2 uint64, s *gosym.Sym) string {
	if val1 == 0 {
		return "= nil"
	}
	sh := &reflect.SliceHeader{
		Data: uintptr(val1),
		Len:  int(val2),
		Cap:  int(val2),
	}
	val := reflect.NewAt(reflectType(s.GoType), unsafe.Pointer(sh))
	return fmt.Sprintf("= %v", val.Elem().Interface())
}

func emptyInterfaceRepr(val1 uint64, val2 uint64) string {
	if val1 == 0 || val2 == 0 {
		return pointerRepr(val2, nil, false)
	}
	i := *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: uintptr(val1),
		val: uintptr(val2),
	}))
	return fmt.Sprintf("= %T(%v)", i, i)
}

func typeName(table *gosym.Table, fn *gosym.Func, s *gosym.Sym) string {
	return reflectType(s.GoType).String()
}

func isInterface(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, tn string) bool {
	i := *(*interface{})(unsafe.Pointer(&s.GoType))
	return reflect.TypeOf(i).Kind() == reflect.Interface
}
