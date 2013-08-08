package sql

import (
	"fmt"
	"gondola/types"
	"reflect"
	"time"
)

type backendScanner struct {
	Out     *reflect.Value
	Tag     *types.Tag
	Nil     bool
	Backend Backend
}

var backendScannerPool = make(chan *backendScanner, 64)

// Always assume the type is right
func (s *backendScanner) Scan(src interface{}) error {
	switch x := src.(type) {
	case nil:
		// Assign zero to the type
		s.Nil = true
		s.Out.Set(reflect.Zero(s.Out.Type()))
		return nil
	case int64:
		return s.Backend.ScanInt(x, s.Out, s.Tag)
	case float64:
		return s.Backend.ScanFloat(x, s.Out, s.Tag)
	case bool:
		return s.Backend.ScanBool(x, s.Out, s.Tag)
	case []byte:
		s.Nil = len(x) == 0
		return s.Backend.ScanByteSlice(x, s.Out, s.Tag)
	case string:
		return s.Backend.ScanString(x, s.Out, s.Tag)
	case time.Time:
		return s.Backend.ScanTime(&x, s.Out, s.Tag)
	}
	return fmt.Errorf("can't scan value %v (%T)", src)
}

func (s *backendScanner) IsNil() bool {
	return s.Nil
}

func (s *backendScanner) Put() {
	select {
	case backendScannerPool <- s:
	default:
	}
}

func BackendScanner(val *reflect.Value, t *types.Tag, backend Backend) scanner {
	var s *backendScanner
	select {
	case s = <-backendScannerPool:
		s.Out = val
		s.Tag = t
		s.Nil = false
		s.Backend = backend
	default:
		s = &backendScanner{Out: val, Tag: t, Backend: backend}
	}
	return s
}
