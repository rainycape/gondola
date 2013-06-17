package driver

import (
	"reflect"
)

type Methods struct {
	// The index for the Load method. -1 if there's no Load method
	LoadIndex int
	// The index for the Save method. -1 if there's no Load method
	SaveIndex int
}

func (m *Methods) Load(obj interface{}) error {
	return m.method(m.LoadIndex, obj)
}

func (m *Methods) Save(obj interface{}) error {
	return m.method(m.SaveIndex, obj)
}

func (m *Methods) method(idx int, obj interface{}) error {
	if idx >= 0 {
		val := reflect.ValueOf(obj)
		for val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
			val = val.Elem()
		}
		ret := val.Method(idx).Call(nil)
		if len(ret) > 0 {
			err, _ := ret[0].Interface().(error)
			return err
		}
	}
	return nil
}

func MakeMethods(typ reflect.Type) (m Methods, err error) {
	m = Methods{-1, -1}
	// Get pointer methods
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	// Check for Load and Save methods
	if load, ok := typ.MethodByName("Load"); ok {
		if err = checkMethod(load); err != nil {
			return
		}
		m.LoadIndex = load.Index
	}
	if save, ok := typ.MethodByName("Save"); ok {
		if err = checkMethod(save); err != nil {
			return
		}
		m.SaveIndex = save.Index
	}
	return
}
