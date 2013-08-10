// +build !appengine

package binary

import (
	"errors"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"sync"
	"unsafe"
)

var decoders struct {
	sync.RWMutex
	cache map[reflect.Type]typeDecoder
}

type typeDecoder func(dec *decoder, p unsafe.Pointer) error

type decoder struct {
	coder
	io.Reader
}

func skipDecoder(typ reflect.Type) (typeDecoder, error) {
	s, err := sizeof(typ)
	if err != nil {
		return nil, err
	}
	l := int64(s)
	return func(dec *decoder, _ unsafe.Pointer) error {
		_, err := io.CopyN(ioutil.Discard, dec, l)
		return err
	}, nil
}

func sliceDecoder(typ reflect.Type) (typeDecoder, error) {
	etyp := typ.Elem()
	switch etyp.Kind() {
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
		// Take advantage of the fast path in Read
		return func(dec *decoder, p unsafe.Pointer) error {
			v := reflect.NewAt(typ, p).Elem()
			return Read(dec, dec.order, v.Interface())
		}, nil
	}
	edec, err := makeDecoder(typ.Elem())
	if err != nil {
		return nil, err
	}
	s := etyp.Size()
	return func(dec *decoder, p unsafe.Pointer) error {
		h := (*reflect.SliceHeader)(p)
		ep := unsafe.Pointer(h.Data)
		for ii := 0; ii < h.Len; ii++ {
			if err := edec(dec, ep); err != nil {
				return err
			}
			ep = unsafe.Pointer(uintptr(ep) + s)
		}
		return nil
	}, nil
}

func arrayDecoder(typ reflect.Type) (typeDecoder, error) {
	etyp := typ.Elem()
	al := typ.Len()
	switch etyp.Kind() {
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
		// Take advantage of the fast path in Read
		return func(dec *decoder, p unsafe.Pointer) error {
			v := reflect.NewAt(typ, p).Elem().Slice(0, al)
			return Read(dec, dec.order, v.Interface())
		}, nil
	}
	edec, err := makeDecoder(typ.Elem())
	if err != nil {
		return nil, err
	}
	s := etyp.Size()
	return func(dec *decoder, p unsafe.Pointer) error {
		for ii := 0; ii < al; ii++ {
			if err := edec(dec, p); err != nil {
				return err
			}
			p = unsafe.Pointer(uintptr(p) + s)
		}
		return nil
	}, nil
}

func structDecoder(typ reflect.Type) (typeDecoder, error) {
	var decoders []typeDecoder
	var offsets []uintptr
	count := typ.NumField()
	var dec typeDecoder
	var err error
	for ii := 0; ii < count; ii++ {
		f := typ.Field(ii)
		ftyp := f.Type
		if f.Name == "_" {
			dec, err = skipDecoder(ftyp)
		} else {
			if f.PkgPath != "" {
				continue
			}
			dec, err = makeDecoder(ftyp)
		}
		if err != nil {
			return nil, err
		}
		decoders = append(decoders, dec)
		offsets = append(offsets, f.Offset)
	}
	return func(dec *decoder, p unsafe.Pointer) error {
		for ii, fdec := range decoders {
			fp := unsafe.Pointer(uintptr(p) + offsets[ii])
			if err := fdec(dec, fp); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func int8Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:1]
	if err := readAtLeast(dec, bs, 1); err != nil {
		return err
	}
	v := (*uint8)(p)
	*v = bs[0]
	return nil
}

func int16Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:2]
	if err := readAtLeast(dec, bs, 2); err != nil {
		return err
	}
	v := (*uint16)(p)
	*v = dec.order.Uint16(bs)
	return nil
}

func int32Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:4]
	if err := readAtLeast(dec, bs, 4); err != nil {
		return err
	}
	v := (*uint32)(p)
	*v = dec.order.Uint32(bs)
	return nil
}

func int64Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v := (*uint64)(p)
	*v = dec.order.Uint64(bs)
	return nil
}

func float32Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:4]
	if err := readAtLeast(dec, bs, 4); err != nil {
		return err
	}
	v := (*float32)(p)
	*v = math.Float32frombits(dec.order.Uint32(bs))
	return nil
}

func float64Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v := (*float64)(p)
	*v = math.Float64frombits(dec.order.Uint64(bs))
	return nil
}

func complex64Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v := (*complex64)(p)
	*v = complex(
		math.Float32frombits(dec.order.Uint32(bs)),
		math.Float32frombits(dec.order.Uint32(bs[4:])),
	)
	return nil
}

func complex128Decoder(dec *decoder, p unsafe.Pointer) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	f1 := math.Float64frombits(dec.order.Uint64(bs))
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v := (*complex128)(p)
	*v = complex(f1, math.Float64frombits(dec.order.Uint64(bs)))
	return nil
}

func newDecoder(typ reflect.Type) (typeDecoder, error) {
	switch typ.Kind() {
	case reflect.Array:
		return arrayDecoder(typ)
	case reflect.Slice:
		return sliceDecoder(typ)
	case reflect.Struct:
		return structDecoder(typ)
	case reflect.Int8, reflect.Uint8:
		return int8Decoder, nil
	case reflect.Int16, reflect.Uint16:
		return int16Decoder, nil
	case reflect.Int32, reflect.Uint32:
		return int32Decoder, nil
	case reflect.Int64, reflect.Uint64:
		return int64Decoder, nil

	case reflect.Float32:
		return float32Decoder, nil
	case reflect.Float64:
		return float64Decoder, nil

	case reflect.Complex64:
		return complex64Decoder, nil
	case reflect.Complex128:
		return complex128Decoder, nil
	}
	return nil, errors.New("can't decode type " + typ.String())
}

func makeDecoder(typ reflect.Type) (typeDecoder, error) {
	decoders.RLock()
	decoder := decoders.cache[typ]
	decoders.RUnlock()
	if decoder == nil {
		var err error
		decoder, err = newDecoder(typ)
		if err != nil {
			return nil, err
		}
		decoders.Lock()
		if decoders.cache == nil {
			decoders.cache = map[reflect.Type]typeDecoder{}
		}
		decoders.cache[typ] = decoder
		decoders.Unlock()
	}
	return decoder, nil
}

func valueDecoder(data interface{}) (unsafe.Pointer, typeDecoder, error) {
	var v reflect.Value
	switch d := reflect.ValueOf(data); d.Kind() {
	case reflect.Ptr:
		v = d.Elem()
	case reflect.Slice:
		v = d
	case reflect.Invalid:
		return nil, nil, errors.New("can't decode into nil")
	default:
		return nil, nil, errors.New("invalid type " + d.Type().String())
	}
	dec, err := makeDecoder(v.Type())
	if err != nil {
		return nil, nil, err
	}
	return unsafe.Pointer(v.UnsafeAddr()), dec, err
}
