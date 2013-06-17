package driver

import (
	"fmt"
	"reflect"
)

func checkMethod(typ reflect.Type, m reflect.Method) error {
	if m.Type.NumIn() != 1 {
		return fmt.Errorf("method %q on type %v should receive no arguments", m.Name, typ)
	}
	if out := m.Type.NumOut(); out > 0 {
		if out > 1 {
			return fmt.Errorf("method %q on type %v may return only 1 or 0 arguments", m.Name, typ)
		}
		if m.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			return fmt.Errorf("method %q on type %v can only return error (it returns %v)", m.Name, typ, m.Type.Out(0))
		}
	}
	return nil
}
