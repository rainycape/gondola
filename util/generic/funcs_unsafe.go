// +build !appengine

package generic

import (
	"reflect"
	"unsafe"
)

type handle uintptr

func indexer(t reflect.Type) indexFunc {
	size := t.Size()
	return func(v handle, i int) handle {
		header := (*reflect.SliceHeader)(unsafe.Pointer(&v))
		return handle(header.Data + uintptr(i)*size)
	}
}

func swapper(t reflect.Type) swapFunc {
	size := t.Size()
	tmp := make([]byte, size)
	return func(v handle, i, j int) {
		header := (*reflect.SliceHeader)(unsafe.Pointer(&v))
		var si []byte
		hi := (*reflect.SliceHeader)(unsafe.Pointer(&si))
		hi.Len = int(size)
		hi.Data = header.Data + uintptr(i)*size
		var sj []byte
		hj := (*reflect.SliceHeader)(unsafe.Pointer(&sj))
		hj.Len = int(size)
		hj.Data = header.Data + uintptr(j)*size
		copy(tmp, sj)
		copy(sj, si)
		copy(si, tmp)
	}
}

func fieldValueFunc(field reflect.StructField, depth int) mapFunc {
	offset := field.Offset
	switch depth {
	case 0:
		return func(v handle) handle {
			return handle(uintptr(v) + offset)
		}
	case 1:
		return func(v handle) handle {
			p := (*unsafe.Pointer)(unsafe.Pointer(v))
			return handle(uintptr(*p) + offset)
		}
	default:
		depth--
		return func(v handle) handle {
			p := (*unsafe.Pointer)(unsafe.Pointer(v))
			for ii := 0; ii < depth; ii++ {
				p = (*unsafe.Pointer)(*p)
			}
			return handle(uintptr(*p) + offset)
		}
	}
}

func methodValueFunc(m reflect.Method) mapFunc {
	ptr := m.Func.Pointer()
	p := &ptr
	fn := (*mapFunc)(unsafe.Pointer(&p))
	return *fn
}

func boolLess(a handle, b handle) bool {
	return !*(*bool)(unsafe.Pointer(a)) && *(*bool)(unsafe.Pointer(b))
}

func int8Less(a handle, b handle) bool {
	return *(*int8)(unsafe.Pointer(a)) < *(*int8)(unsafe.Pointer(b))
}

func int16Less(a handle, b handle) bool {
	return *(*int16)(unsafe.Pointer(a)) < *(*int16)(unsafe.Pointer(b))
}

func int32Less(a handle, b handle) bool {
	return *(*int32)(unsafe.Pointer(a)) < *(*int32)(unsafe.Pointer(b))
}

func int64Less(a handle, b handle) bool {
	return *(*int64)(unsafe.Pointer(a)) < *(*int64)(unsafe.Pointer(b))
}

func intLess(a handle, b handle) bool {
	return *(*int)(unsafe.Pointer(a)) < *(*int)(unsafe.Pointer(b))
}

func uint8Less(a handle, b handle) bool {
	return *(*uint8)(unsafe.Pointer(a)) < *(*uint8)(unsafe.Pointer(b))
}

func uint16Less(a handle, b handle) bool {
	return *(*uint16)(unsafe.Pointer(a)) < *(*uint16)(unsafe.Pointer(b))
}

func uint32Less(a handle, b handle) bool {
	return *(*uint32)(unsafe.Pointer(a)) < *(*uint32)(unsafe.Pointer(b))
}

func uint64Less(a handle, b handle) bool {
	return *(*uint64)(unsafe.Pointer(a)) < *(*uint64)(unsafe.Pointer(b))
}

func uintLess(a handle, b handle) bool {
	return *(*uint)(unsafe.Pointer(a)) < *(*uint)(unsafe.Pointer(b))
}

func float32Less(a handle, b handle) bool {
	return *(*float32)(unsafe.Pointer(a)) < *(*float32)(unsafe.Pointer(b))
}

func float64Less(a handle, b handle) bool {
	return *(*float64)(unsafe.Pointer(a)) < *(*float64)(unsafe.Pointer(b))
}

func stringLess(a handle, b handle) bool {
	return *(*string)(unsafe.Pointer(a)) < *(*string)(unsafe.Pointer(b))
}

func getHandle(val reflect.Value) handle {
	return handle(val.Pointer())
}

func getElem(h handle) handle {
	p := (*unsafe.Pointer)(unsafe.Pointer(h))
	return handle(*p)
}

func getComparator(t reflect.Type) lessFunc {
	switch t.Kind() {
	case reflect.Bool:
		return boolLess
	case reflect.Int8:
		return int8Less
	case reflect.Int16:
		return int16Less
	case reflect.Int32:
		return int32Less
	case reflect.Int64:
		return int64Less
	case reflect.Int:
		return intLess
	case reflect.Uint8:
		return uint8Less
	case reflect.Uint16:
		return uint16Less
	case reflect.Uint32:
		return uint32Less
	case reflect.Uint64:
		return uint64Less
	case reflect.Uint:
		return uintLess
	case reflect.Float32:
		return float32Less
	case reflect.Float64:
		return float64Less
	case reflect.String:
		return stringLess
	}
	return nil
}
