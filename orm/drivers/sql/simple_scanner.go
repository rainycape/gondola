package sql

import (
	"fmt"
	"reflect"
	"time"
)

type simpleScanner struct {
	Out *reflect.Value
}

var simpleScannerPool = make(chan *simpleScanner, 64)

// Always assume the type is right
func (s *simpleScanner) Scan(src interface{}) error {
	switch x := src.(type) {
	case nil:
		// Assign zero to the type
		s.Out.Set(reflect.Zero(s.Out.Type()))
	case int64:
		s.Out.SetInt(x)
	case bool:
		s.Out.SetBool(x)
	case []byte:
	case string:
		s.Out.SetString(x)
	case time.Time:
		s.Out.Set(reflect.ValueOf(x))
	default:
		return fmt.Errorf("simple can't scan value %v (%T)", src, src)
	}
	return nil
}

func (s *simpleScanner) Put() {
	select {
	case simpleScannerPool <- s:
	default:
	}
}

func Scanner(val *reflect.Value) scanner {
	var s *simpleScanner
	select {
	case s = <-simpleScannerPool:
		s.Out = val
	default:
		s = &simpleScanner{Out: val}
	}
	return s
}
