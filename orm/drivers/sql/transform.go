package sql

import (
	"reflect"
)

type Transform interface {
	Transform() error
}

type stringTransform struct {
	In  interface{}
	Out reflect.Value
}

func (s *stringTransform) Transform() error {
	switch v := (s.In).(type) {
	case []byte:
		// It sucks to copy this, because database/sql is copying
		// it already.
		s.Out.SetString(string(v))
	}
	return nil
}

type transform struct {
	In      interface{}
	Out     reflect.Value
	Backend Backend
}

func (t *transform) Transform() error {
	return t.Backend.TransformInValue(t.In, t.Out)
}
