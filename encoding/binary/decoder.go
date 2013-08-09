package binary

import (
	"io"
	"math"
	"reflect"
)

type decoder struct {
	coder
	reader io.Reader
}

func (d *decoder) read(bs []byte) {
	if _, err := d.reader.Read(bs); err != nil {
		panic(err)
	}
}

func (d *decoder) uint8() uint8 {
	bs := d.buf[:1]
	d.read(bs)
	return bs[0]
}

func (d *decoder) uint16() uint16 {
	bs := d.buf[:2]
	d.read(bs)
	return d.order.Uint16(bs)
}

func (d *decoder) uint32() uint32 {
	bs := d.buf[:4]
	d.read(bs)
	return d.order.Uint32(bs)
}

func (d *decoder) uint64() uint64 {
	bs := d.buf[:8]
	d.read(bs)
	return d.order.Uint64(bs)
}

func (d *decoder) int8() int8 { return int8(d.uint8()) }

func (d *decoder) int16() int16 { return int16(d.uint16()) }

func (d *decoder) int32() int32 { return int32(d.uint32()) }

func (d *decoder) int64() int64 { return int64(d.uint64()) }

func (d *decoder) value(v reflect.Value) {
	switch v.Kind() {
	case reflect.Array:
		l := v.Len()
		for i := 0; i < l; i++ {
			d.value(v.Index(i))
		}

	case reflect.Struct:
		t := v.Type()
		l := v.NumField()
		for i := 0; i < l; i++ {
			// Note: Calling v.CanSet() below is an optimization.
			// It would be sufficient to check the field name,
			// but creating the StructField info for each field is
			// costly (run "go test -bench=ReadStruct" and compare
			// results when making changes to this code).
			if v := v.Field(i); v.CanSet() || t.Field(i).Name != "_" {
				d.value(v)
			} else {
				d.skip(v)
			}
		}

	case reflect.Slice:
		// Fast path for basic slice types
		switch s := v.Interface().(type) {
		case []int8:
			for i := range s {
				s[i] = d.int8()
			}
		case []uint8:
			for i := range s {
				s[i] = d.uint8()
			}
		case []int16:
			for i := range s {
				s[i] = d.int16()
			}
		case []uint16:
			for i := range s {
				s[i] = d.uint16()
			}
		case []int32:
			for i := range s {
				s[i] = d.int32()
			}
		case []uint32:
			for i := range s {
				s[i] = d.uint32()
			}
		case []int64:
			for i := range s {
				s[i] = d.int64()
			}
		case []uint64:
			for i := range s {
				s[i] = d.uint64()
			}
		default:
			l := v.Len()
			for i := 0; i < l; i++ {
				d.value(v.Index(i))
			}
		}

	case reflect.Int8:
		v.SetInt(int64(d.int8()))
	case reflect.Int16:
		v.SetInt(int64(d.int16()))
	case reflect.Int32:
		v.SetInt(int64(d.int32()))
	case reflect.Int64:
		v.SetInt(d.int64())

	case reflect.Uint8:
		v.SetUint(uint64(d.uint8()))
	case reflect.Uint16:
		v.SetUint(uint64(d.uint16()))
	case reflect.Uint32:
		v.SetUint(uint64(d.uint32()))
	case reflect.Uint64:
		v.SetUint(d.uint64())

	case reflect.Float32:
		v.SetFloat(float64(math.Float32frombits(d.uint32())))
	case reflect.Float64:
		v.SetFloat(math.Float64frombits(d.uint64()))

	case reflect.Complex64:
		v.SetComplex(complex(
			float64(math.Float32frombits(d.uint32())),
			float64(math.Float32frombits(d.uint32())),
		))
	case reflect.Complex128:
		v.SetComplex(complex(
			math.Float64frombits(d.uint64()),
			math.Float64frombits(d.uint64()),
		))
	}
}

func (d *decoder) skip(v reflect.Value) {
	n, _ := dataSize(v)
	b := d.buf[:8]
	for n >= 8 {
		d.read(b)
		n -= 8
	}
	if n > 0 {
		d.read(b[:n])
	}
}
