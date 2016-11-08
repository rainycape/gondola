package kvs

import "testing"

type keyValueStorage map[interface{}]interface{}

func (k keyValueStorage) Get(key interface{}) interface{} {
	return k[key]
}

func (k keyValueStorage) Set(key interface{}, value interface{}) {
	k[key] = value
}

func makeKeyValueStorage() keyValueStorage {
	m := make(map[interface{}]interface{})
	return keyValueStorage(m)
}

func TestKeyValueType(t *testing.T) {
	var (
		intPtrGetter func(Storage) *int
		intPtrSetter func(Storage, *int)
	)
	TypeFuncs(&intPtrGetter, &intPtrSetter)
	kv := makeKeyValueStorage()
	if v := intPtrGetter(kv); v != nil {
		t.Errorf("expecting nil, got %v", v)
	}
	val := 42
	intPtrSetter(kv, &val)
	if v := intPtrGetter(kv); v == nil || *v != val {
		t.Errorf("expecting pointer to %v, got %v", val, v)
	}
	intPtrSetter(kv, nil)
	if v := intPtrGetter(kv); v != nil {
		t.Errorf("expecting nil, got %v", v)
	}
}

func TestKeyValueKey(t *testing.T) {
	const (
		key = "uintptr"
	)
	var (
		intPtrGetter func(Storage, interface{}) *int
		intPtrSetter func(Storage, interface{}, *int)
	)
	Funcs(&intPtrGetter, &intPtrSetter)
	kv := makeKeyValueStorage()
	if v := intPtrGetter(kv, key); v != nil {
		t.Errorf("expecting nil, got %v", v)
	}
	val := 42
	intPtrSetter(kv, key, &val)
	if v := intPtrGetter(kv, key); v == nil || *v != val {
		t.Errorf("expecting pointer to %v, got %v", val, v)
	}
	intPtrSetter(kv, key, nil)
	if v := intPtrGetter(kv, key); v != nil {
		t.Errorf("expecting nil, got %v", v)
	}
}

func TestKVS(t *testing.T) {
	var (
		intGetter func(*KVS) int
		intSetter func(*KVS, int)
	)
	TypeFuncs(&intGetter, &intSetter)
	kv := new(KVS)
	if v := intGetter(kv); v != 0 {
		t.Errorf("expecting 0, got %v", v)
	}
	val := 42
	intSetter(kv, val)
	if v := intGetter(kv); v != val {
		t.Errorf("expecting %v, got %v", val, v)
	}
	intSetter(kv, 0)
	if v := intGetter(kv); v != 0 {
		t.Errorf("expecting 0, got %v", v)
	}
}

func TestBadFunctions(t *testing.T) {
	runFailureTest := func(f func()) (err error) {
		defer func() {
			err = recover().(error)
		}()
		f()
		return
	}
	testFailure := func(f func()) {
		if err := runFailureTest(f); err == nil {
			t.Errorf("expecting error, got nil running %v", f)
		} else {
			t.Logf("got expected error %v", err)
		}
	}
	testFailure(func() {
		var (
			intGetter func(KVS) int
			intSetter func(KVS, int)
		)
		TypeFuncs(&intGetter, &intSetter)
	})
}

func TestCopyClear(t *testing.T) {
	var (
		intGetter func(*KVS) int
		intSetter func(*KVS, int)
	)
	TypeFuncs(&intGetter, &intSetter)
	kv := new(KVS)
	val := 42
	intSetter(kv, val)
	if v := intGetter(kv); v != val {
		t.Errorf("expecting %v, got %v", val, v)
	}
	cpy := kv.Copy()
	kv.Clear()
	if v := intGetter(kv); v != 0 {
		t.Errorf("expecting %v, got %v", 0, v)
	}
	if v := intGetter(cpy); v != val {
		t.Errorf("expecting %v, got %v", val, v)
	}
}
