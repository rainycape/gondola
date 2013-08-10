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
	"sync"
)

var sizes struct {
	sync.RWMutex
	cache map[reflect.Type]int
}

// Size returns how many bytes Write would generate to encode the value v, which
// must be a fixed-size value or a slice of fixed-size values, or a pointer to such data.
func Size(v interface{}) int {
	n, err := dataSize(reflect.Indirect(reflect.ValueOf(v)).Type())
	if err != nil {
		return -1
	}
	return n
}

// dataSize returns the number of bytes the actual data represented by t occupies in memory.
// For compound structures, it sums the sizes of the elements. Thus, for instance, for a slice
// it returns the length of the slice times the element size and does not count the memory
// occupied by the header.
func dataSize(typ reflect.Type) (int, error) {
	sizes.RLock()
	size, ok := sizes.cache[typ]
	sizes.RUnlock()
	if !ok {
		var err error
		if size, err = sizeof(typ); err != nil {
			return size, err
		}
		sizes.Lock()
		if sizes.cache == nil {
			sizes.cache = make(map[reflect.Type]int)
		}
		sizes.cache[typ] = size
		sizes.Unlock()
	}
	return size, nil
}

func sizeof(t reflect.Type) (int, error) {
	switch t.Kind() {
	case reflect.Array, reflect.Slice:
		n, err := sizeof(t.Elem())
		if err != nil {
			return 0, err
		}
		return t.Len() * n, nil

	case reflect.Struct:
		sum := 0
		for i, n := 0, t.NumField(); i < n; i++ {
			s, err := dataSize(t.Field(i).Type)
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
