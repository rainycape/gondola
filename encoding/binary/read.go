package binary

import (
	"errors"
	"io"
	"reflect"
)

// readAtLeast is an optimized version of io.ReadAtLeast,
// which omits some checks that don't need to be performed
// when called from Read() in this package.
func readAtLeast(r io.Reader, buf []byte, min int) error {
	var n int
	var err error
	// Most common case, we get all the bytes in one read
	if n, err = r.Read(buf); n == min {
		return nil
	}
	if err != nil {
		return err
	}
	// Fall back to looping
	var nn int
	for n < min {
		nn, err = r.Read(buf[n:])
		if err != nil {
			if err == io.EOF && n > 0 {
				err = io.ErrUnexpectedEOF
			}
			return err
		}
		n += nn
	}
	return nil
}

// Read reads structured binary data from r into data.
// Data must be a pointer to a fixed-size value or a slice
// of fixed-size values.
// Bytes read from r are decoded using the specified byte order
// and written to successive fields of the data.
// When reading into structs, the field data for fields with
// blank (_) field names is skipped; i.e., blank field names
// may be used for padding.
func Read(r io.Reader, order ByteOrder, data interface{}) error {
	// Fast path for basic types and slices of basic types
	var err error
	switch v := data.(type) {
	case *int8:
		bs := make([]byte, 1)
		if err = readAtLeast(r, bs, 1); err != nil {
			return err
		}
		*v = int8(bs[0])
		return nil
	case *uint8:
		bs := make([]byte, 1)
		if err = readAtLeast(r, bs, 1); err != nil {
			return err
		}
		*v = bs[0]
		return nil
	case *int16:
		bs := make([]byte, 2)
		if err = readAtLeast(r, bs, 2); err != nil {
			return err
		}
		*v = int16(order.Uint16(bs))
		return nil
	case *uint16:
		bs := make([]byte, 2)
		if err = readAtLeast(r, bs, 2); err != nil {
			return err
		}
		*v = order.Uint16(bs)
		return nil
	case *int32:
		bs := make([]byte, 4)
		if err = readAtLeast(r, bs, 4); err != nil {
			return err
		}
		*v = int32(order.Uint32(bs))
		return nil
	case *uint32:
		bs := make([]byte, 4)
		if err = readAtLeast(r, bs, 4); err != nil {
			return err
		}
		*v = order.Uint32(bs)
		return nil
	case *int64:
		bs := make([]byte, 8)
		if err = readAtLeast(r, bs, 8); err != nil {
			return err
		}
		*v = int64(order.Uint64(bs))
		return nil
	case *uint64:
		bs := make([]byte, 8)
		if err = readAtLeast(r, bs, 8); err != nil {
			return err
		}
		*v = order.Uint64(bs)
		return nil
	case []int8:
		bs := make([]byte, 8)
		count := len(v)
		steps := count / 8
		i := 0
		for j := 0; j < steps; j++ {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = int8(bs[0])
			i++
			v[i] = int8(bs[1])
			i++
			v[i] = int8(bs[2])
			i++
			v[i] = int8(bs[3])
			i++
			v[i] = int8(bs[4])
			i++
			v[i] = int8(bs[5])
			i++
			v[i] = int8(bs[6])
			i++
			v[i] = int8(bs[7])
			i++
		}
		if i < count {
			rem := count - i
			br := bs[:rem]
			if err = readAtLeast(r, br, rem); err != nil {
				return err
			}
			for j := 0; j < rem; j++ {
				v[i] = int8(br[j])
				i++
			}
		}
		return nil
	case []uint8:
		if err = readAtLeast(r, v, len(v)); err != nil {
			return err
		}
		return nil
	case []int16:
		bs := make([]byte, 8)
		count := len(v)
		steps := count / 4
		i := 0
		for j := 0; j < steps; j++ {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = int16(order.Uint16(bs))
			i++
			v[i] = int16(order.Uint16(bs[2:]))
			i++
			v[i] = int16(order.Uint16(bs[4:]))
			i++
			v[i] = int16(order.Uint16(bs[6:]))
			i++
		}
		if i < count {
			rem := (count - i) * 2
			br := bs[:rem]
			if err = readAtLeast(r, br, rem); err != nil {
				return err
			}
			for j := 0; j < rem; j += 2 {
				v[i] = int16(order.Uint16(br[j : j+2]))
				i++
			}
		}
		return nil
	case []uint16:
		bs := make([]byte, 8)
		count := len(v)
		steps := count / 4
		i := 0
		for j := 0; j < steps; j++ {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = order.Uint16(bs)
			i++
			v[i] = order.Uint16(bs[2:])
			i++
			v[i] = order.Uint16(bs[4:])
			i++
			v[i] = order.Uint16(bs[6:])
			i++
		}
		if i < count {
			rem := (count - i) * 2
			br := bs[:rem]
			if err = readAtLeast(r, br, rem); err != nil {
				return err
			}
			for j := 0; j < rem; j += 2 {
				v[i] = order.Uint16(br[j : j+2])
				i++
			}
		}
		return nil
	case []int32:
		bs := make([]byte, 8)
		count := len(v)
		steps := count / 2
		i := 0
		for j := 0; j < steps; j++ {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = int32(order.Uint32(bs))
			i++
			v[i] = int32(order.Uint32(bs[4:]))
			i++
		}
		if i != count {
			b4 := bs[:4]
			if err = readAtLeast(r, b4, 4); err != nil {
				return err
			}
			v[i] = int32(order.Uint32(b4))
		}
		return nil
	case []uint32:
		bs := make([]byte, 8)
		count := len(v)
		steps := count / 2
		i := 0
		for j := 0; j < steps; j++ {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = order.Uint32(bs)
			i++
			v[i] = order.Uint32(bs[4:])
			i++
		}
		if i != count {
			b4 := bs[:4]
			if err = readAtLeast(r, b4, 4); err != nil {
				return err
			}
			v[i] = order.Uint32(b4)
		}
		return nil
	case []int64:
		bs := make([]byte, 8)
		for i := range v {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = int64(order.Uint64(bs))
		}
		return nil
	case []uint64:
		bs := make([]byte, 8)
		for i := range v {
			if err = readAtLeast(r, bs, 8); err != nil {
				return err
			}
			v[i] = order.Uint64(bs)
		}
		return nil
	}

	// Fallback to reflect-based decoding.
	var v reflect.Value
	switch d := reflect.ValueOf(data); d.Kind() {
	case reflect.Ptr:
		v = d.Elem()
	case reflect.Slice:
		v = d
	default:
		return errors.New("binary.Read: invalid type " + d.Type().String())
	}
	if _, err := dataSize(v); err != nil {
		return errors.New("binary.Read: " + err.Error())
	}
	d := decoder{coder: coder{order: order}, reader: r}
	d.value(v)
	return nil
}
