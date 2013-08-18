package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"gondola/html"
	"strconv"
	"strings"
)

func basicType(typ string) bool {
	switch typ {
	case "bool", "int", "uint", "byte", "string",
		"int8", "uint8", "int16", "uint16",
		"int32", "uint32", "int64", "uint64",
		"float32", "float64", "complex64", "complex128":
		return true
	}
	return false
}

func interfaceRepr(val1 uint64, val2 uint64) string {
	return pointerRepr(val2, nil, false)
}

func valRepr(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, tn string, values []string, _html bool) (string, int) {
	val, _ := strconv.ParseUint(values[0], 0, 64)
	if basicType(tn) {
		switch {
		case tn == "bool":
			if val == 0 {
				return "= false", 1
			}
			return "= true", 1
		case strings.HasPrefix(tn, "int"):
			return "= " + strconv.FormatInt(int64(val), 10), 1
		case strings.HasPrefix(tn, "uint") || tn == "byte":
			return "= " + strconv.FormatUint(val, 10), 1
		case tn == "float32":
		case tn == "float64":
		case tn == "string":
			s := stringRepr(val)
			if _html {
				s = html.Escape(s)
			}
			return s, 1
		}
	}
	if len(values) > 1 && values[1] != "..." {
		val2, _ := strconv.ParseUint(values[1], 0, 64)
		if tn == "interface {}" {
			return emptyInterfaceRepr(val, val2), 2
		}
		if isInterface(table, fn, s, tn) {
			return interfaceRepr(val, val2), 2
		}
	}
	return pointerRepr(val, s, _html), 1
}

func fieldRepr(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, values []string, _html bool) (repr string, used int, ok bool) {
	tn := typeName(table, fn, s)
	if tn == "" {
		return
	}
	ok = true
	name := s.Name[strings.IndexByte(s.Name, '.')+1:]
	var rep string
	rep, used = valRepr(table, fn, s, tn, values, _html)
	if rep != "" {
		repr = fmt.Sprintf("%s %s %s", name, tn, rep)
	} else {
		repr = fmt.Sprintf("%s %s", name, tn)
	}
	return
}
