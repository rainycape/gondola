package binary

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
)

var encoders struct {
	sync.RWMutex
	cache map[reflect.Type]typeEncoder
}

type typeEncoder func(enc *encoder, v reflect.Value) error

type encoder struct {
	coder
	writer io.Writer
}

func (e *encoder) write(bs []byte) error {
	_, err := e.writer.Write(bs)
	return err
}

func skipEncoder(typ reflect.Type) (typeEncoder, error) {
	s, err := dataSize(typ)
	if err != nil {
		return nil, err
	}
	return func(enc *encoder, v reflect.Value) error {
		for ii := 0; ii < 8; ii++ {
			enc.buf[ii] = 0
		}
		b := enc.buf[:8]
		n := s
		for n >= 8 {
			if err := enc.write(b); err != nil {
				return err
			}
			n -= 8
		}
		if n > 0 {
			return enc.write(b[:n])
		}
		return nil
	}, nil
}

func sliceEncoder(typ reflect.Type) (typeEncoder, error) {
	eenc, err := makeEncoder(typ.Elem())
	if err != nil {
		return nil, err
	}
	return func(enc *encoder, v reflect.Value) error {
		for ii := 0; ii < v.Len(); ii++ {
			if err := eenc(enc, v.Index(ii)); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func structEncoder(typ reflect.Type) (typeEncoder, error) {
	var encoders []typeEncoder
	var indexes [][]int
	count := typ.NumField()
	var enc typeEncoder
	var err error
	for ii := 0; ii < count; ii++ {
		f := typ.Field(ii)
		ftyp := f.Type
		if f.Name == "_" {
			enc, err = skipEncoder(ftyp)
		} else {
			if f.PkgPath != "" {
				continue
			}
			enc, err = makeEncoder(ftyp)
		}
		if err != nil {
			return nil, err
		}
		encoders = append(encoders, enc)
		indexes = append(indexes, f.Index)
	}
	return func(enc *encoder, v reflect.Value) error {
		for ii, fenc := range encoders {
			f := v.FieldByIndex(indexes[ii])
			if err := fenc(enc, f); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func int8Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:1]
	bs[0] = byte(int8(v.Int()))
	return enc.write(bs)
}

func int16Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:2]
	enc.order.PutUint16(bs, uint16(v.Int()))
	return enc.write(bs)
}

func int32Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:4]
	enc.order.PutUint32(bs, uint32(v.Int()))
	return enc.write(bs)
}

func int64Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:8]
	enc.order.PutUint64(bs, uint64(v.Int()))
	return enc.write(bs)
}

func uint8Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:1]
	bs[0] = uint8(v.Uint())
	return enc.write(bs)
}

func uint16Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:2]
	enc.order.PutUint16(bs, uint16(v.Uint()))
	return enc.write(bs)
}

func uint32Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:4]
	enc.order.PutUint32(bs, uint32(v.Uint()))
	return enc.write(bs)
}

func uint64Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:8]
	enc.order.PutUint64(bs, uint64(v.Uint()))
	return enc.write(bs)
}

func float32Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:4]
	enc.order.PutUint32(bs, math.Float32bits(float32(v.Float())))
	return enc.write(bs)
}

func float64Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:8]
	enc.order.PutUint64(bs, math.Float64bits(v.Float()))
	return enc.write(bs)
}

func complex64Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:8]
	x := v.Complex()
	enc.order.PutUint32(bs, math.Float32bits(float32(real(x))))
	enc.order.PutUint32(bs[4:], math.Float32bits(float32(imag(x))))
	return enc.write(bs)
}

func complex128Encoder(enc *encoder, v reflect.Value) error {
	bs := enc.buf[:8]
	x := v.Complex()
	enc.order.PutUint64(bs, math.Float64bits(real(x)))
	if err := enc.write(bs); err != nil {
		return err
	}
	enc.order.PutUint64(bs, math.Float64bits(imag(x)))
	return enc.write(bs)
}

func newEncoder(typ reflect.Type) (typeEncoder, error) {
	switch typ.Kind() {
	case reflect.Array, reflect.Slice:
		return sliceEncoder(typ)
	case reflect.Struct:
		return structEncoder(typ)
	case reflect.Int8:
		return int8Encoder, nil
	case reflect.Int16:
		return int16Encoder, nil
	case reflect.Int32:
		return int32Encoder, nil
	case reflect.Int64:
		return int64Encoder, nil

	case reflect.Uint8:
		return uint8Encoder, nil
	case reflect.Uint16:
		return uint16Encoder, nil
	case reflect.Uint32:
		return uint32Encoder, nil
	case reflect.Uint64:
		return uint64Encoder, nil

	case reflect.Float32:
		return float32Encoder, nil
	case reflect.Float64:
		return float64Encoder, nil

	case reflect.Complex64:
		return complex64Encoder, nil
	case reflect.Complex128:
		return complex128Encoder, nil
	}
	return nil, fmt.Errorf("can't encode type %v", typ)
}

func makeEncoder(typ reflect.Type) (typeEncoder, error) {
	encoders.RLock()
	encoder := encoders.cache[typ]
	encoders.RUnlock()
	if encoder == nil {
		var err error
		encoder, err = newEncoder(typ)
		if err != nil {
			return nil, err
		}
		encoders.Lock()
		if encoders.cache == nil {
			encoders.cache = map[reflect.Type]typeEncoder{}
		}
		encoders.cache[typ] = encoder
		encoders.Unlock()
	}
	return encoder, nil
}
