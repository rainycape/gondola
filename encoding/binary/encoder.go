package binary

import (
	"io"
	"math"
	"reflect"
)

type encoder struct {
	coder
	writer io.Writer
}

func (e *encoder) write(bs []byte) {
	if _, err := e.writer.Write(bs); err != nil {
		panic(err)
	}
}

func (e *encoder) uint8(x uint8) {
	bs := e.buf[:1]
	bs[0] = x
	e.write(bs)
}

func (e *encoder) uint16(x uint16) {
	bs := e.buf[:2]
	e.order.PutUint16(bs, x)
	e.write(bs)
}

func (e *encoder) uint32(x uint32) {
	bs := e.buf[:4]
	e.order.PutUint32(bs, x)
	e.write(bs)
}

func (e *encoder) uint64(x uint64) {
	bs := e.buf[:8]
	e.order.PutUint64(bs, x)
	e.write(bs)
}

func (e *encoder) int8(x int8) { e.uint8(uint8(x)) }

func (e *encoder) int16(x int16) { e.uint16(uint16(x)) }

func (e *encoder) int32(x int32) { e.uint32(uint32(x)) }

func (e *encoder) int64(x int64) { e.uint64(uint64(x)) }

func (e *encoder) value(v reflect.Value) {
	switch v.Kind() {
	case reflect.Array:
		l := v.Len()
		for i := 0; i < l; i++ {
			e.value(v.Index(i))
		}

	case reflect.Struct:
		t := v.Type()
		l := v.NumField()
		for i := 0; i < l; i++ {
			// see comment for corresponding code in decoder.value()
			if v := v.Field(i); v.CanSet() || t.Field(i).Name != "_" {
				e.value(v)
			} else {
				e.skip(v)
			}
		}

	case reflect.Slice:
		// Fast path for basic slice types
		switch s := v.Interface().(type) {
		case []int8:
			for _, val := range s {
				e.int8(val)
			}
		case []uint8:
			for _, val := range s {
				e.uint8(val)
			}
		case []int16:
			for _, val := range s {
				e.int16(val)
			}
		case []uint16:
			for _, val := range s {
				e.uint16(val)
			}
		case []int32:
			for _, val := range s {
				e.int32(val)
			}
		case []uint32:
			for _, val := range s {
				e.uint32(val)
			}
		case []int64:
			for _, val := range s {
				e.int64(val)
			}
		case []uint64:
			for _, val := range s {
				e.uint64(val)
			}
		default:
			l := v.Len()
			for i := 0; i < l; i++ {
				e.value(v.Index(i))
			}
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v.Type().Kind() {
		case reflect.Int8:
			e.int8(int8(v.Int()))
		case reflect.Int16:
			e.int16(int16(v.Int()))
		case reflect.Int32:
			e.int32(int32(v.Int()))
		case reflect.Int64:
			e.int64(v.Int())
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch v.Type().Kind() {
		case reflect.Uint8:
			e.uint8(uint8(v.Uint()))
		case reflect.Uint16:
			e.uint16(uint16(v.Uint()))
		case reflect.Uint32:
			e.uint32(uint32(v.Uint()))
		case reflect.Uint64:
			e.uint64(v.Uint())
		}

	case reflect.Float32, reflect.Float64:
		switch v.Type().Kind() {
		case reflect.Float32:
			e.uint32(math.Float32bits(float32(v.Float())))
		case reflect.Float64:
			e.uint64(math.Float64bits(v.Float()))
		}

	case reflect.Complex64, reflect.Complex128:
		switch v.Type().Kind() {
		case reflect.Complex64:
			x := v.Complex()
			e.uint32(math.Float32bits(float32(real(x))))
			e.uint32(math.Float32bits(float32(imag(x))))
		case reflect.Complex128:
			x := v.Complex()
			e.uint64(math.Float64bits(real(x)))
			e.uint64(math.Float64bits(imag(x)))
		}
	}
}

func (e *encoder) skip(v reflect.Value) {
	n, _ := dataSize(v.Type())
	for ii := 0; ii < 8; ii++ {
		e.buf[ii] = 0
	}
	b := e.buf[:8]
	for n >= 8 {
		e.write(b)
		n -= 8
	}
	if n > 0 {
		e.write(b[:n])
	}
}
