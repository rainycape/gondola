package generic

import (
	"reflect"
)

func boolReflectLess(a *reflect.Value, b *reflect.Value) bool {
	return !a.Bool() && b.Bool()
}

func intReflectLess(a *reflect.Value, b *reflect.Value) bool {
	return a.Int() < b.Int()
}

func uintReflectLess(a *reflect.Value, b *reflect.Value) bool {
	return a.Uint() < b.Uint()
}

func floatReflectLess(a *reflect.Value, b *reflect.Value) bool {
	return a.Float() < b.Float()
}

func stringReflectLess(a *reflect.Value, b *reflect.Value) bool {
	return a.String() < b.String()
}

func sliceReflectLess(a *reflect.Value, b *reflect.Value) bool {
	return a.Len() < b.Len()
}

func getReflectComparator(t reflect.Type) func(*reflect.Value, *reflect.Value) bool {
	switch t.Kind() {
	case reflect.Bool:
		return boolReflectLess
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return intReflectLess
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return uintReflectLess
	case reflect.Float32, reflect.Float64:
		return floatReflectLess
	case reflect.String:
		return stringReflectLess
	case reflect.Array, reflect.Slice:
		return sliceReflectLess
	}
	return nil
}

func intReflectAdd(a *reflect.Value, b *reflect.Value) reflect.Value {
	return reflect.ValueOf(a.Int() + b.Int())
}

func uintReflectAdd(a *reflect.Value, b *reflect.Value) reflect.Value {
	return reflect.ValueOf(a.Uint() + b.Uint())
}

func floatReflectAdd(a *reflect.Value, b *reflect.Value) reflect.Value {
	return reflect.ValueOf(a.Float() + b.Float())
}

func stringReflectAdd(a *reflect.Value, b *reflect.Value) reflect.Value {
	return reflect.ValueOf(a.String() + b.String())
}

func sliceReflectAdd(a *reflect.Value, b *reflect.Value) reflect.Value {
	total := a.Len() + b.Len()
	s := reflect.MakeSlice(a.Type(), total, total)
	ii := 0
	for jj := 0; jj < a.Len(); jj++ {
		s.Index(ii).Set(a.Index(jj))
		ii++
	}
	for jj := 0; jj < b.Len(); jj++ {
		s.Index(ii).Set(b.Index(jj))
		ii++
	}
	return s
}

func getReflectAdder(t reflect.Type) func(*reflect.Value, *reflect.Value) reflect.Value {
	switch t.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return intReflectAdd
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return uintReflectAdd
	case reflect.Float32, reflect.Float64:
		return floatReflectAdd
	case reflect.String:
		return stringReflectAdd
	case reflect.Array, reflect.Slice:
		return sliceReflectAdd
	}
	return nil
}
