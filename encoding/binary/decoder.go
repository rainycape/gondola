// +build appengine

package binary

import (
	"errors"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"sync"
)

var decoders struct {
	sync.RWMutex
	cache map[reflect.Type]typeDecoder
}

type typeDecoder func(dec *decoder, v reflect.Value) error

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
	return func(dec *decoder, v reflect.Value) error {
		_, err := io.CopyN(ioutil.Discard, dec, l)
		return err
	}, nil
}

func sliceDecoder(typ reflect.Type) (typeDecoder, error) {
	switch typ.Elem().Kind() {
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
		// Take advantage of the fast path in Read
		if typ.Kind() == reflect.Slice {
			return func(dec *decoder, v reflect.Value) error {
				return Read(dec, dec.order, v.Interface())
			}, nil
		}
		// Array
		al := typ.Len()
		return func(dec *decoder, v reflect.Value) error {
			// Value must be addressable when we reach this point
			return Read(dec, dec.order, v.Slice(0, al).Interface())
		}, nil
	}
	edec, err := makeDecoder(typ.Elem())
	if err != nil {
		return nil, err
	}
	return func(dec *decoder, v reflect.Value) error {
		sl := v.Len()
		for ii := 0; ii < sl; ii++ {
			if err := edec(dec, v.Index(ii)); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func structDecoder(typ reflect.Type) (typeDecoder, error) {
	var decoders []typeDecoder
	var indexes [][]int
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
		indexes = append(indexes, f.Index)
	}
	return func(dec *decoder, v reflect.Value) error {
		for ii, fdec := range decoders {
			f := v.FieldByIndex(indexes[ii])
			if err := fdec(dec, f); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func int8Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:1]
	if err := readAtLeast(dec, bs, 1); err != nil {
		return err
	}
	v.SetInt(int64(bs[0]))
	return nil
}

func int16Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:2]
	if err := readAtLeast(dec, bs, 2); err != nil {
		return err
	}
	v.SetInt(int64(dec.order.Uint16(bs)))
	return nil
}

func int32Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:4]
	if err := readAtLeast(dec, bs, 4); err != nil {
		return err
	}
	v.SetInt(int64(dec.order.Uint32(bs)))
	return nil
}

func int64Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v.SetInt(int64(dec.order.Uint64(bs)))
	return nil
}

func uint8Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:1]
	if err := readAtLeast(dec, bs, 1); err != nil {
		return err
	}
	v.SetUint(uint64(bs[0]))
	return nil
}

func uint16Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:2]
	if err := readAtLeast(dec, bs, 2); err != nil {
		return err
	}
	v.SetUint(uint64(dec.order.Uint16(bs)))
	return nil
}

func uint32Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:4]
	if err := readAtLeast(dec, bs, 4); err != nil {
		return err
	}
	v.SetUint(uint64(dec.order.Uint32(bs)))
	return nil
}

func uint64Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v.SetUint(uint64(dec.order.Uint64(bs)))
	return nil
}

func float32Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:4]
	if err := readAtLeast(dec, bs, 4); err != nil {
		return err
	}
	v.SetFloat(float64(math.Float32frombits(dec.order.Uint32(bs))))
	return nil
}

func float64Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v.SetFloat(float64(math.Float64frombits(dec.order.Uint64(bs))))
	return nil
}

func complex64Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v.SetComplex(complex(
		float64(math.Float32frombits(dec.order.Uint32(bs))),
		float64(math.Float32frombits(dec.order.Uint32(bs[4:]))),
	))
	return nil
}

func complex128Decoder(dec *decoder, v reflect.Value) error {
	bs := dec.buf[:8]
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	f1 := math.Float64frombits(dec.order.Uint64(bs))
	if err := readAtLeast(dec, bs, 8); err != nil {
		return err
	}
	v.SetComplex(complex(f1, math.Float64frombits(dec.order.Uint64(bs))))
	return nil
}

func newDecoder(typ reflect.Type) (typeDecoder, error) {
	switch typ.Kind() {
	case reflect.Array, reflect.Slice:
		return sliceDecoder(typ)
	case reflect.Struct:
		return structDecoder(typ)
	case reflect.Int8:
		return int8Decoder, nil
	case reflect.Int16:
		return int16Decoder, nil
	case reflect.Int32:
		return int32Decoder, nil
	case reflect.Int64:
		return int64Decoder, nil

	case reflect.Uint8:
		return uint8Decoder, nil
	case reflect.Uint16:
		return uint16Decoder, nil
	case reflect.Uint32:
		return uint32Decoder, nil
	case reflect.Uint64:
		return uint64Decoder, nil

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

func valueDecoder(data interface{}) (reflect.Value, typeDecoder, error) {
	var v reflect.Value
	switch d := reflect.ValueOf(data); d.Kind() {
	case reflect.Ptr:
		v = d.Elem()
	case reflect.Slice:
		v = d
	case reflect.Invalid:
		return reflect.Value{}, nil, errors.New("can't decode into nil")
	default:
		return reflect.Value{}, nil, errors.New("invalid type " + d.Type().String())
	}
	dec, err := makeDecoder(v.Type())
	if err != nil {
		return reflect.Value{}, nil, err
	}
	return v, dec, err
}
