package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"gondola/html"
	"math"
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

func valRepr(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, tn string, values []string, _html bool) (r string) {
	val, _ := strconv.ParseUint(values[0], 0, 64)
	// If there's a panic prettyfy'ing the value just
	// assume it's a pointer. It's better than
	// omitting the error page.
	defer func() {
		if recover() != nil {
			r = pointerRepr(val, s, false)
		}
	}()
	if basicType(tn) {
		switch {
		case tn == "bool":
			if val == 0 {
				return "= false"
			}
			return "= true"
		case strings.HasPrefix(tn, "int"):
			return "= " + strconv.FormatInt(int64(val), 10)
		case strings.HasPrefix(tn, "uint") || tn == "byte":
			return "= " + strconv.FormatUint(val, 10)
		case tn == "float32":
			return "= " + strconv.FormatFloat(float64(math.Float32frombits(uint32(val))), 'g', -1, 32)
		case tn == "float64":
			return "= " + strconv.FormatFloat(math.Float64frombits(uint64(val)), 'g', -1, 64)
		}
	}
	if len(values) > 1 && values[1] != "..." {
		val2, _ := strconv.ParseUint(values[1], 0, 64)
		if tn == "string" {
			v := stringRepr(val, val2)
			if _html {
				v = html.Escape(v)
			}
			return v
		}
		if tn == "interface {}" {
			return emptyInterfaceRepr(val, val2)
		}
		if isInterface(table, fn, s, tn) {
			return interfaceRepr(val, val2)
		}
	}
	return pointerRepr(val, s, _html)
}

func fieldRepr(table *gosym.Table, fn *gosym.Func, s *gosym.Sym, values []string, _html bool) (repr string, ok bool) {
	tn := typeName(table, fn, s)
	if tn == "" {
		return
	}
	ok = true
	name := s.BaseName()
	var rep string
	rep = valRepr(table, fn, s, tn, values, _html)
	if rep != "" {
		repr = fmt.Sprintf("%s %s %s", name, tn, rep)
	} else {
		repr = fmt.Sprintf("%s %s", name, tn)
	}
	return
}
