// +build !appengine

package binary

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
	"unsafe"
)

var encoders struct {
	sync.RWMutex
	cache map[reflect.Type]typeEncoder
}

type typeEncoder func(enc *encoder, p unsafe.Pointer) error

type encoder struct {
	coder
	io.Writer
}

func skipEncoder(typ reflect.Type) (typeEncoder, error) {
	s, err := sizeof(typ)
	if err != nil {
		return nil, err
	}
	return func(enc *encoder, _ unsafe.Pointer) error {
		for ii := 0; ii < 8; ii++ {
			enc.buf[ii] = 0
		}
		b := enc.buf[:8]
		n := s
		for n >= 8 {
			if _, err := enc.Write(b); err != nil {
				return err
			}
			n -= 8
		}
		if n > 0 {
			_, err := enc.Write(b[:n])
			return err
		}
		return nil
	}, nil
}

func sliceEncoder(typ reflect.Type) (typeEncoder, error) {
	switch typ.Elem().Kind() {
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
		// Take advantage of the fast path in Write
		return func(enc *encoder, p unsafe.Pointer) error {
			v := reflect.NewAt(typ, p).Elem()
			return Write(enc, enc.order, v.Interface())
		}, nil
	}
	etyp := typ.Elem()
	eenc, err := makeEncoder(etyp)
	if err != nil {
		return nil, err
	}
	s := etyp.Size()
	return func(enc *encoder, p unsafe.Pointer) error {
		h := (*reflect.SliceHeader)(p)
		ep := unsafe.Pointer(h.Data)
		for ii := 0; ii < h.Len; ii++ {
			if err := eenc(enc, ep); err != nil {
				return err
			}
		}
		ep = unsafe.Pointer(uintptr(ep) + s)
		return nil
	}, nil
}

func arrayEncoder(typ reflect.Type) (typeEncoder, error) {
	al := typ.Len()
	etyp := typ.Elem()
	switch etyp.Kind() {
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
		// Take advantage of the fast path in Write
		return func(enc *encoder, p unsafe.Pointer) error {
			v := reflect.NewAt(typ, p).Elem().Slice(0, al)
			return Write(enc, enc.order, v.Interface())
		}, nil
	}
	eenc, err := makeEncoder(etyp)
	if err != nil {
		return nil, err
	}
	s := etyp.Size()
	return func(enc *encoder, p unsafe.Pointer) error {
		for ii := 0; ii < al; ii++ {
			if err := eenc(enc, p); err != nil {
				return err
			}
			p = unsafe.Pointer(uintptr(p) + s)
		}
		return nil
	}, nil
}

func structEncoder(typ reflect.Type) (typeEncoder, error) {
	var encoders []typeEncoder
	var offsets []uintptr
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
		offsets = append(offsets, f.Offset)
	}
	return func(enc *encoder, p unsafe.Pointer) error {
		for ii, fenc := range encoders {
			fp := unsafe.Pointer(uintptr(p) + offsets[ii])
			if err := fenc(enc, fp); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func int8Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:1]
	v := (*uint8)(p)
	bs[0] = *v
	_, err := enc.Write(bs)
	return err
}

func int16Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:2]
	v := (*uint16)(p)
	enc.order.PutUint16(bs, *v)
	_, err := enc.Write(bs)
	return err
}

func int32Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:4]
	v := (*uint32)(p)
	enc.order.PutUint32(bs, *v)
	_, err := enc.Write(bs)
	return err
}

func int64Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:8]
	v := (*uint64)(p)
	enc.order.PutUint64(bs, *v)
	_, err := enc.Write(bs)
	return err
}

func float32Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:4]
	v := (*float32)(p)
	enc.order.PutUint32(bs, math.Float32bits(*v))
	_, err := enc.Write(bs)
	return err
}

func float64Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:8]
	v := (*float64)(p)
	enc.order.PutUint64(bs, math.Float64bits(*v))
	_, err := enc.Write(bs)
	return err
}

func complex64Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:8]
	v := (*complex64)(p)
	enc.order.PutUint32(bs, math.Float32bits(real(*v)))
	enc.order.PutUint32(bs[4:], math.Float32bits(imag(*v)))
	_, err := enc.Write(bs)
	return err
}

func complex128Encoder(enc *encoder, p unsafe.Pointer) error {
	bs := enc.buf[:8]
	v := (*complex128)(p)
	enc.order.PutUint64(bs, math.Float64bits(real(*v)))
	if _, err := enc.Write(bs); err != nil {
		return err
	}
	enc.order.PutUint64(bs, math.Float64bits(imag(*v)))
	_, err := enc.Write(bs)
	return err
}

func newEncoder(typ reflect.Type) (typeEncoder, error) {
	switch typ.Kind() {
	case reflect.Array:
		return arrayEncoder(typ)
	case reflect.Slice:
		return sliceEncoder(typ)
	case reflect.Struct:
		return structEncoder(typ)
	case reflect.Int8, reflect.Uint8:
		return int8Encoder, nil
	case reflect.Int16, reflect.Uint16:
		return int16Encoder, nil
	case reflect.Int32, reflect.Uint32:
		return int32Encoder, nil
	case reflect.Int64, reflect.Uint64:
		return int64Encoder, nil

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

type emptyInterface struct {
	typ  unsafe.Pointer
	word unsafe.Pointer
}

func valueEncoder(data interface{}) (unsafe.Pointer, typeEncoder, error) {
	v := reflect.Indirect(reflect.ValueOf(data))
	enc, err := makeEncoder(v.Type())
	if err != nil {
		return nil, nil, err
	}
	var p unsafe.Pointer
	if v.CanAddr() {
		p = unsafe.Pointer(v.UnsafeAddr())
	} else {
		i := (*emptyInterface)(unsafe.Pointer(&data))
		p = i.word
	}
	return p, enc, err
}
