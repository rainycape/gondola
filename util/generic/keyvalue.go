package generic

import (
	"fmt"
	"reflect"
)

var (
	keyValueStorageType = reflect.TypeOf((*KeyValueStorage)(nil)).Elem()
	stringType          = reflect.TypeOf("")
)

// KeyValueStorage is an interface which declares two methods for
// storing arbitrary values. The lifetime of the values as well of
// the thread safety of the storage is dependent on the implementation.
type KeyValueStorage interface {
	Get(key string) interface{}
	Set(key string, value interface{})
}

func keyValueGet(kv KeyValueStorage, key string, typ reflect.Type) []reflect.Value {
	v := kv.Get(key)
	var rv reflect.Value
	if v == nil {
		// We've got an untyped nil, so we need to
		// create a typed one with the return value
		// of the function.
		rv = reflect.Zero(typ)
	} else {
		rv = reflect.ValueOf(v)
	}
	return []reflect.Value{rv}
}

func keyValueStorageGet(key string, typ reflect.Type) func([]reflect.Value) []reflect.Value {
	return func(in []reflect.Value) []reflect.Value {
		kv := in[0].Interface().(KeyValueStorage)
		return keyValueGet(kv, key, typ)
	}
}

func keyValueStorageGetKey(typ reflect.Type) func([]reflect.Value) []reflect.Value {
	return func(in []reflect.Value) []reflect.Value {
		kv := in[0].Interface().(KeyValueStorage)
		key := in[1].String()
		return keyValueGet(kv, key, typ)
	}
}

func keyValueStorageSet(key string) func([]reflect.Value) []reflect.Value {
	return func(in []reflect.Value) []reflect.Value {
		kv := in[0].Interface().(KeyValueStorage)
		kv.Set(key, in[1].Interface())
		return nil
	}
}

func keyValueStorageSetKey(in []reflect.Value) []reflect.Value {
	kv := in[0].Interface().(KeyValueStorage)
	key := in[1].String()
	kv.Set(key, in[2].Interface())
	return nil
}

func isKeyValueStorageType(typ reflect.Type) bool {
	return typ == keyValueStorageType || typ.AssignableTo(keyValueStorageType)
}

// MakeKeyValueFuncs allows creating functions for easily setting and retrieving
// values from a KeyValueStorage (like gnd.la/app.App and gnd.la/app.Context) in
// a type safe manner. For a given type T you should declare to vars like
//
//      var (
//          Getter func(generic.KeyValueStorage, string) T
//          Setter func(generic.KeyValueStorage, string, T)
//      )
//
// Then from an init() function call MakeKeyValueFuncs like this:
//
//      generic.MakeKeyValueFuncs(&Getter, &Setter)
//
// Note that if the prototypes of the functions don't match the expected
// ones, this function will panic.
//
// Alternatively, if you're only going to store once instance per type, it's
// usually better to use MakeKeyValueTypeFuncs.
func MakeKeyValueFuncs(getter interface{}, setter interface{}) {
	gptr := reflect.ValueOf(getter)
	if gptr.Kind() != reflect.Ptr || gptr.Elem().Kind() != reflect.Func {
		panic(fmt.Errorf("getter must be a pointer to a function, not %v",
			gptr.Type()))
	}
	gval := gptr.Elem()
	gvalType := gval.Type()
	if gvalType.NumIn() != 2 {
		panic(fmt.Errorf("getter must accept two arguments, not %d",
			gvalType.NumIn()))
	}
	if !isKeyValueStorageType(gvalType.In(0)) {
		panic(fmt.Errorf("getter 1st argument must be of type %v or assignable to it, not %v",
			keyValueStorageType, gvalType.In(0)))
	}
	if gvalType.In(1) != stringType {
		panic(fmt.Errorf("getter 2nd argument must be of type %v, not %v",
			stringType, gvalType.In(1)))
	}
	if gvalType.NumOut() != 1 {
		panic(fmt.Errorf("getter must return only one value, not %d",
			gvalType.NumOut()))
	}
	ttype := gvalType.Out(0)
	gval.Set(reflect.MakeFunc(gvalType, keyValueStorageGetKey(ttype)))

	sptr := reflect.ValueOf(setter)
	if sptr.Kind() != reflect.Ptr || sptr.Elem().Kind() != reflect.Func {
		panic(fmt.Errorf("setter must be a pointer to a function, not %v",
			sptr.Type()))
	}
	sval := sptr.Elem()
	svalType := sval.Type()
	if svalType.NumIn() != 3 {
		panic(fmt.Errorf("setter must accept three arguments, not %d",
			svalType.NumIn()))
	}
	if !isKeyValueStorageType(svalType.In(0)) {
		panic(fmt.Errorf("setter's 1st argument must be of type %v or assignable to it, not %v",
			keyValueStorageType, svalType.In(0)))
	}
	if svalType.In(1) != stringType {
		panic(fmt.Errorf("setter's 2nd argument must be of type %v, not %v",
			stringType, svalType.In(1)))
	}
	if svalType.In(2) != ttype {
		panic(fmt.Errorf("setter's 3rd argument must be of type %v (to match getter), not %v",
			ttype, svalType.In(2)))
	}
	if svalType.NumOut() != 0 {
		panic(fmt.Errorf("setter not return any values, not %d",
			svalType.NumOut()))
	}
	sval.Set(reflect.MakeFunc(svalType, keyValueStorageSetKey))
}

// MakeKeyValueTypeFuncs allows creating functions for easily setting and retrieving
// values from a KeyValueStorage (like gnd.la/app.App and gnd.la/app.Context) in
// a type safe manner. For a given type T you should declare to vars like
//
//      var (
//          Getter func(generic.KeyValueStorage) T
//          Setter func(generic.KeyValueStorage, T)
//      )
//
// Then from an init() function call MakeKeyValueFuncs like this:
//
//      generic.MakeKeyValueTypeFuncs(&Getter, &Setter)
//
// Note that if the prototypes of the functions don't match the expected
// ones, this function will panic.
//
// Alternatively, if you need to get/set multiple instances of the same type, you
// can use MakeKeyValueFuncs instead.
func MakeKeyValueTypeFuncs(getter interface{}, setter interface{}) {
	gptr := reflect.ValueOf(getter)
	if gptr.Kind() != reflect.Ptr || gptr.Elem().Kind() != reflect.Func {
		panic(fmt.Errorf("getter must be a pointer to a function, not %v",
			gptr.Type()))
	}
	gval := gptr.Elem()
	gvalType := gval.Type()
	if gvalType.NumIn() != 1 {
		panic(fmt.Errorf("getter must accept only one argument, not %d",
			gvalType.NumIn()))
	}
	if !isKeyValueStorageType(gvalType.In(0)) {
		panic(fmt.Errorf("getter 1st argument must be of type %v or assignable to it, not %v",
			keyValueStorageType, gvalType.In(0)))
	}
	if gvalType.NumOut() != 1 {
		panic(fmt.Errorf("getter must return only one value, not %d",
			gvalType.NumOut()))
	}
	ttype := gvalType.Out(0)
	key := fmt.Sprintf("%p-%s", getter, ttype.String())
	gval.Set(reflect.MakeFunc(gvalType, keyValueStorageGet(key, ttype)))

	sptr := reflect.ValueOf(setter)
	if sptr.Kind() != reflect.Ptr || sptr.Elem().Kind() != reflect.Func {
		panic(fmt.Errorf("setter must be a pointer to a function, not %v",
			sptr.Type()))
	}
	sval := sptr.Elem()
	svalType := sval.Type()
	if svalType.NumIn() != 2 {
		panic(fmt.Errorf("setter must accept two arguments, not %d",
			svalType.NumIn()))
	}
	if !isKeyValueStorageType(svalType.In(0)) {
		panic(fmt.Errorf("setter's 1st argument must be of type %v or assignable to it, not %v",
			keyValueStorageType, svalType.In(0)))
	}
	if svalType.In(1) != ttype {
		panic(fmt.Errorf("setter's 2nd argument must be of type %v (to match getter), not %v",
			ttype, svalType.In(1)))
	}
	if svalType.NumOut() != 0 {
		panic(fmt.Errorf("setter not return any values, not %d",
			svalType.NumOut()))
	}
	sval.Set(reflect.MakeFunc(svalType, keyValueStorageSet(key)))
}
