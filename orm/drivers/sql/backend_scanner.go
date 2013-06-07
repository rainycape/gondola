package sql

import (
	"fmt"
	"reflect"
	"time"
)

type backendScanner struct {
	Out     *reflect.Value
	Backend Backend
}

var backendScannerPool = make(chan *backendScanner, 64)

// Always assume the type is right
func (s *backendScanner) Scan(src interface{}) error {
	switch x := src.(type) {
	case nil:
		// Assign zero to the type
		s.Out.Set(reflect.Zero(s.Out.Type()))
		return nil
	case int64:
		return s.Backend.ScanInt(x, s.Out)
	case bool:
		return s.Backend.ScanBool(x, s.Out)
	case []byte:
		return s.Backend.ScanByteSlice(x, s.Out)
	case string:
		return s.Backend.ScanString(x, s.Out)
	case time.Time:
		return s.Backend.ScanTime(&x, s.Out)
	}
	return fmt.Errorf("can't scan value %v (%T)", src)
}

func (s *backendScanner) Put() {
	select {
	case backendScannerPool <- s:
	default:
	}
}

func BackendScanner(val *reflect.Value, backend Backend) scanner {
	var s *backendScanner
	select {
	case s = <-backendScannerPool:
		s.Out = val
		s.Backend = backend
	default:
		s = &backendScanner{Out: val, Backend: backend}
	}
	return s
}
