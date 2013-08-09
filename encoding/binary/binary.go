// Copyright 2009 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package binary implements simple translation between numbers and byte
// sequences and encoding and decoding of varints.
//
// Numbers are translated by reading and writing fixed-size values.
// A fixed-size value is either a fixed-size arithmetic
// type (int8, uint8, int16, float32, complex64, ...)
// or an array or struct containing only fixed-size values.
//
// Varints are a method of encoding integers using one or more bytes;
// numbers with smaller absolute value take a smaller number of bytes.
// For a specification, see http://code.google.com/apis/protocolbuffers/docs/encoding.html.
//
package binary

import (
	"errors"
	"reflect"
)

// Size returns how many bytes Write would generate to encode the value v, which
// must be a fixed-size value or a slice of fixed-size values, or a pointer to such data.
func Size(v interface{}) int {
	n, err := dataSize(reflect.Indirect(reflect.ValueOf(v)))
	if err != nil {
		return -1
	}
	return n
}

// dataSize returns the number of bytes the actual data represented by v occupies in memory.
// For compound structures, it sums the sizes of the elements. Thus, for instance, for a slice
// it returns the length of the slice times the element size and does not count the memory
// occupied by the header.
func dataSize(v reflect.Value) (int, error) {
	if v.Kind() == reflect.Slice {
		elem, err := sizeof(v.Type().Elem())
		if err != nil {
			return 0, err
		}
		return v.Len() * elem, nil
	}
	return sizeof(v.Type())
}

func sizeof(t reflect.Type) (int, error) {
	switch t.Kind() {
	case reflect.Array:
		n, err := sizeof(t.Elem())
		if err != nil {
			return 0, err
		}
		return t.Len() * n, nil

	case reflect.Struct:
		sum := 0
		for i, n := 0, t.NumField(); i < n; i++ {
			s, err := sizeof(t.Field(i).Type)
			if err != nil {
				return 0, err
			}
			sum += s
		}
		return sum, nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return int(t.Size()), nil
	}
	return 0, errors.New("invalid type " + t.String())
}
