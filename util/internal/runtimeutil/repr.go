// +build !windows,!appengine

package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"gnd.la/html"
	"gnd.la/util/types"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"unsafe"
)

func valRepr(s *gosym.Sym, typ reflect.Type, values []string, _html bool) (r string) {
	val, _ := strconv.ParseUint(values[0], 0, 64)
	var val2 uint64
	if len(values) > 1 {
		val2, _ = strconv.ParseUint(values[1], 0, 64)
	}
	// If there's a panic prettyfy'ing the value just
	// assume it's a pointer. It's better than
	// omitting the error page.
	defer func() {
		if recover() != nil {
			r = pointerRepr(nil, val, false)
		}
	}()
	switch types.Kind(typ.Kind()) {
	case types.Bool:
		if val == 0 {
			return "= false"
		}
		return "= true"
	case types.Int:
		return "= " + strconv.FormatInt(int64(val), 10)
	case types.Uint:
		return "= " + strconv.FormatUint(val, 10)
	case types.Float:
		if typ.Kind() == reflect.Float32 {
			return "= " + strconv.FormatFloat(float64(math.Float32frombits(uint32(val))), 'g', -1, 32)
		}
		return "= " + strconv.FormatFloat(math.Float64frombits(uint64(val)), 'g', -1, 64)
	case types.Slice:
		return sliceRepr(val, val2, s)
	case types.String:
		v := stringRepr(val, val2)
		if _html {
			v = html.Escape(v)
		}
		return v
	case types.Interface:
		if typ.NumMethod() == 0 {
			return emptyInterfaceRepr(val, val2)
		}
		idata := [2]uintptr{uintptr(val), uintptr(val2)}
		v := reflect.NewAt(typ, unsafe.Pointer(&idata[0])).Elem()
		return descRepr(val, &v, _html)
	case types.Func:
		fn := reflect.NewAt(typ, unsafe.Pointer(&val)).Elem()
		f := runtime.FuncForPC(fn.Pointer())
		if f != nil {
			return "= " + f.Name()
		}
	}
	return pointerRepr(typ, val, _html)
}

func fieldRepr(s *gosym.Sym, typ reflect.Type, values []string, _html bool) (string, bool) {
	name := s.BaseName()
	repr := valRepr(s, typ, values, _html)
	if repr == "" {
		return "", false
	}
	return fmt.Sprintf("%s %s %s", name, typ, repr), true
}

type emptyInterface struct {
	typ uintptr
	val uintptr
}

func pointerRepr(typ reflect.Type, val uint64, _html bool) string {
	if val == 0 {
		return "= nil"
	}
	if typ != nil && _html {
		v := reflect.NewAt(typ, unsafe.Pointer(&val)).Elem()
		if typ.Kind() == reflect.Map || (typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct) {
			return descRepr(val, &v, _html)
		}
	}
	return descRepr(val, nil, _html)
}

func descRepr(p uint64, val *reflect.Value, _html bool) string {
	var typeName string
	isInterface := val != nil && val.Kind() == reflect.Interface
	if isInterface {
		typeName = html.Escape(fmt.Sprintf("%T", val.Interface()))
	}
	ptr := fmt.Sprintf("0x%x", p)
	if val != nil && _html {
		title := fmt.Sprintf("%+v", val.Interface())
		if isInterface {
			return fmt.Sprintf("@ %s(<abbr title=\"%s\">%s</abbr>)", typeName, html.Escape(title), ptr)
		}
		return fmt.Sprintf("@ <abbr title=\"%s\">%s</abbr>", html.Escape(title), ptr)
	}
	if isInterface {
		return fmt.Sprintf("@ %s(%s)", typeName, ptr)
	}
	return fmt.Sprintf("@ %s", ptr)
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
		return pointerRepr(nil, val2, false)
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
