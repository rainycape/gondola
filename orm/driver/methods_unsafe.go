// +build !appengine

package driver

import (
	"reflect"
	"unsafe"
)

type method func(uintptr) error

type Methods struct {
	// The address for the Load method. 0 if there's no Load method
	LoadPointer unsafe.Pointer
	// Wheter Load returns an error
	LoadReturns bool
	// The address for the Save method. 0 if there's no Save method
	SavePointer unsafe.Pointer
	// Wheter Save returns an error
	SaveReturns bool
}

func (m *Methods) Load(obj interface{}) error {
	return m.method(m.LoadPointer, m.LoadReturns, obj)
}

func (m *Methods) Save(obj interface{}) error {
	return m.method(m.SavePointer, m.SaveReturns, obj)
}

func (m *Methods) method(p unsafe.Pointer, ret bool, obj interface{}) error {
	if p != nil {
		val := reflect.ValueOf(obj)
		for val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
			val = val.Elem()
		}
		f := *(*method)(unsafe.Pointer(&p))
		if ret {
			return f(val.Pointer())
		}
		f(val.Pointer())
	}
	return nil
}

func MakeMethods(typ reflect.Type) (m *Methods, err error) {
	m = &Methods{}
	// Get pointer methods
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	// Check for Load and Save methods
	if load, ok := typ.MethodByName("Load"); ok {
		if err = checkMethod(typ, load); err != nil {
			return
		}
		m.LoadPointer = pointer(typ, load.Index)
		m.LoadReturns = returns(load)
	}
	if save, ok := typ.MethodByName("Save"); ok {
		if err = checkMethod(typ, save); err != nil {
			return
		}
		m.SavePointer = pointer(typ, save.Index)
		m.SaveReturns = returns(save)
	}
	return
}

func pointer(typ reflect.Type, idx int) unsafe.Pointer {
	ptr := typ.Method(idx).Func.Pointer()
	return unsafe.Pointer(&ptr)
}

func returns(m reflect.Method) bool {
	return m.Type.NumOut() > 0
}
