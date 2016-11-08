package kvs

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	storageType        = reflect.TypeOf((*Storage)(nil)).Elem()
	emptyInterfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
)

// KVS is a thread safe implementation of the
// Storage interface. Its empty value is safe
// to use.
type KVS struct {
	mu     sync.RWMutex
	values map[interface{}]interface{}
}

// Get implements Storage.Get
func (k *KVS) Get(key interface{}) interface{} {
	k.mu.RLock()
	value := k.values[key]
	k.mu.RUnlock()
	return value
}

// Set implements Storage.Set
func (k *KVS) Set(key interface{}, value interface{}) {
	k.mu.Lock()
	if k.values == nil {
		k.values = make(map[interface{}]interface{})
	}
	k.values[key] = value
	k.mu.Unlock()
}

// Clear removes all stored values
func (k *KVS) Clear() {
	k.mu.Lock()
	k.values = nil
	k.mu.Unlock()
}

// Copy returns a shallow copy of the KVS
func (k *KVS) Copy() *KVS {
	cpy := new(KVS)
	k.mu.RLock()
	if len(k.values) > 0 {
		cpy.values = make(map[interface{}]interface{})
		for k, v := range k.values {
			cpy.values[k] = v
		}
	}
	k.mu.RUnlock()
	return cpy
}

// Storage is an interface which declares two methods for
// storing arbitrary values. The lifetime of the values as well of
// the thread safety of the storage is dependent on the implementation.
type Storage interface {
	Get(key interface{}) interface{}
	Set(key interface{}, value interface{})
}

func keyValueGet(kv Storage, key interface{}, typ reflect.Type) []reflect.Value {
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

func storageGet(key interface{}, typ reflect.Type) func([]reflect.Value) []reflect.Value {
	return func(in []reflect.Value) []reflect.Value {
		kv := in[0].Interface().(Storage)
		return keyValueGet(kv, key, typ)
	}
}

func storageGetKey(typ reflect.Type) func([]reflect.Value) []reflect.Value {
	return func(in []reflect.Value) []reflect.Value {
		kv := in[0].Interface().(Storage)
		key := in[1].Interface()
		return keyValueGet(kv, key, typ)
	}
}

func storageSet(key interface{}) func([]reflect.Value) []reflect.Value {
	return func(in []reflect.Value) []reflect.Value {
		kv := in[0].Interface().(Storage)
		kv.Set(key, in[1].Interface())
		return nil
	}
}

func storageSetKey(in []reflect.Value) []reflect.Value {
	kv := in[0].Interface().(Storage)
	kv.Set(in[1].Interface(), in[2].Interface())
	return nil
}

func isStorageType(typ reflect.Type) bool {
	return typ == storageType || typ.AssignableTo(storageType)
}

// Funcs allows creating functions for easily setting and retrieving
// values associated with a key from an Storage in a type safe manner.
//
// Getter functions must conform to the following specification:
//
//  func(storage S, key interface{}) V or func(storage S, key interface{}) (V, bool)
//
// Where S is kvs.Storage or implements kvs.Storage and V is any type.
//
// Setter functions must conform to the following specification:
//
//  func(storage S, key interface{}, value V)
//
// Where V is any type. Note that when generating a getter/setter function pair,
// V must be exactly the same type in the getter and in the setter.
//
// See the examples for more information.
//
// Note that this function will panic if the prototypes of the functions don't,
// match the expected ones.
//
// Alternatively, if you're only going to store once value per type, use
// TypeFuncs instead.
func Funcs(getter interface{}, setter interface{}) {
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
	if !isStorageType(gvalType.In(0)) {
		panic(fmt.Errorf("getter 1st argument must be of type %v or assignable to it, not %v",
			storageType, gvalType.In(0)))
	}
	if gvalType.In(1) != emptyInterfaceType {
		panic(fmt.Errorf("getter 2nd argument must be of type %v, not %v",
			emptyInterfaceType, gvalType.In(1)))
	}
	if gvalType.NumOut() != 1 {
		panic(fmt.Errorf("getter must return only one value, not %d",
			gvalType.NumOut()))
	}
	ttype := gvalType.Out(0)
	gval.Set(reflect.MakeFunc(gvalType, storageGetKey(ttype)))

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
	if !isStorageType(svalType.In(0)) {
		panic(fmt.Errorf("setter's 1st argument must be of type %v or assignable to it, not %v",
			storageType, svalType.In(0)))
	}
	if svalType.In(1) != emptyInterfaceType {
		panic(fmt.Errorf("setter's 2nd argument must be of type %v, not %v",
			emptyInterfaceType, svalType.In(1)))
	}
	if svalType.In(2) != ttype {
		panic(fmt.Errorf("setter's 3rd argument must be of type %v (to match getter), not %v",
			ttype, svalType.In(2)))
	}
	if svalType.NumOut() != 0 {
		panic(fmt.Errorf("setter not return any values, not %d",
			svalType.NumOut()))
	}
	sval.Set(reflect.MakeFunc(svalType, storageSetKey))
}

// TypeFuncs allows creating functions for easily setting and retrieving
// a value associated with a type from an Storage in a type safe manner.
//
// Getter functions must conform to the following specification:
//
//  func(storage S) V or func(storage S) (V, bool)
//
// Where S is kvs.Storage or implements kvs.Storage and V is any type.
//
// Setter functions must conform to the following specification:
//
//  func(storage S, value V)
//
// Where V is any type. Note that when generating a getter/setter function pair,
// V must be exactly the same type in the getter and in the setter.
//
// See the examples for more information.
//
// Alternatively, if you need to get/set multiple values of the same type, use
// Funcs instead.
func TypeFuncs(getter interface{}, setter interface{}) {
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
	if !isStorageType(gvalType.In(0)) {
		panic(fmt.Errorf("getter 1st argument must be of type %v or assignable to it, not %v",
			storageType, gvalType.In(0)))
	}
	if gvalType.NumOut() != 1 {
		panic(fmt.Errorf("getter must return only one value, not %d",
			gvalType.NumOut()))
	}
	ttype := gvalType.Out(0)
	key := gval
	gval.Set(reflect.MakeFunc(gvalType, storageGet(key, ttype)))

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
	if !isStorageType(svalType.In(0)) {
		panic(fmt.Errorf("setter's 1st argument must be of type %v or assignable to it, not %v",
			storageType, svalType.In(0)))
	}
	if svalType.In(1) != ttype {
		panic(fmt.Errorf("setter's 2nd argument must be of type %v (to match getter), not %v",
			ttype, svalType.In(1)))
	}
	if svalType.NumOut() != 0 {
		panic(fmt.Errorf("setter not return any values, not %d",
			svalType.NumOut()))
	}
	sval.Set(reflect.MakeFunc(svalType, storageSet(key)))
}
