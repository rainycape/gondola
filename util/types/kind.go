package types

import (
	"reflect"
)

// KindGroup groups different reflect Kinds which
// support the same set of operations
type KindGroup uint

const (
	// Invalid includes reflect.Invalid
	Invalid KindGroup = iota
	// Bool includes reflect.Bool
	Bool
	// Int includes reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32 and reflect.Int64
	Int
	// Uint includes reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32 and reflect.Uint64
	Uint
	// Uintptr includes reflect.Uintptr
	Uintptr
	// Float includes reflect.Float32 and reflect.Float64
	Float
	// Complex includes reflect.Complex64 and reflect.Complex128
	Complex
	// Array includes reflect.Array
	Array
	// Chan includes reflect.Chan
	Chan
	// Func includes reflect.Func
	Func
	// Interface includes reflect.Interface
	Interface
	// Map includes reflect.Map
	Map
	// Ptr includes reflect.Ptr
	Ptr
	// Slice includes reflect.Slice
	Slice
	// String includes reflect.String
	String
	// Struct includes reflect.Struct
	Struct
	// UnsafePointer includes reflect.UnsafePointer
	UnsafePointer
)

// Kind returns the KindGroup for the given reflect.Kind.
func Kind(k reflect.Kind) KindGroup {
	switch k {
	case reflect.Invalid:
		return Invalid
	case reflect.Bool:
		return Bool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Int
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Uint
	case reflect.Uintptr:
		return Uintptr
	case reflect.Float32, reflect.Float64:
		return Float
	case reflect.Complex64, reflect.Complex128:
		return Complex
	case reflect.Array:
		return Array
	case reflect.Chan:
		return Chan
	case reflect.Func:
		return Func
	case reflect.Interface:
		return Interface
	case reflect.Map:
		return Map
	case reflect.Ptr:
		return Ptr
	case reflect.Slice:
		return Slice
	case reflect.String:
		return String
	case reflect.Struct:
		return Struct
	case reflect.UnsafePointer:
		return UnsafePointer
	}
	panic("unreachable")
}
