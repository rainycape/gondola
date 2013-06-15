package sql

import (
	"fmt"
	"gondola/orm/codec"
	"gondola/orm/tag"
	"reflect"
	"time"
)

type simpleScanner struct {
	Out *reflect.Value
	Tag *tag.Tag
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
		if c := codec.FromTag(s.Tag); c != nil {
			return c.Decode(x, s.Out)
		}
		// Some sql drivers return strings as []byte
		if s.Out.Kind() == reflect.String {
			s.Out.SetString(string(x))
			return nil
		}
		// Some drivers return an empty slice for null blob fields
		if len(x) > 0 {
			if !s.Tag.Has("raw") {
				b := make([]byte, len(x))
				copy(b, x)
				x = b
			}
			s.Out.Set(reflect.ValueOf(x))
		} else {
			s.Out.Set(reflect.ValueOf([]byte(nil)))
		}
	case string:
		s.Out.SetString(x)
	case time.Time:
		s.Out.Set(reflect.ValueOf(x))
	default:
		return fmt.Errorf("can't scan value %v (%T)", src, src)
	}
	return nil
}

func (s *simpleScanner) Put() {
	select {
	case simpleScannerPool <- s:
	default:
	}
}

func Scanner(val *reflect.Value, t *tag.Tag) scanner {
	var s *simpleScanner
	select {
	case s = <-simpleScannerPool:
		s.Out = val
		s.Tag = t
	default:
		s = &simpleScanner{Out: val, Tag: t}
	}
	return s
}
