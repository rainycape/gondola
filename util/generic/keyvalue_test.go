package generic

import "testing"

type keyValueStorage map[string]interface{}

func (k keyValueStorage) Get(key string) interface{} {
	return k[key]
}

func (k keyValueStorage) Set(key string, value interface{}) {
	k[key] = value
}

func makeKeyValueStorage() keyValueStorage {
	m := make(map[string]interface{})
	return keyValueStorage(m)
}

func TestKeyValueType(t *testing.T) {
	var (
		intPtrGetter func(KeyValueStorage) *int
		intPtrSetter func(KeyValueStorage, *int)
	)
	MakeKeyValueTypeFuncs(&intPtrGetter, &intPtrSetter)
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
		intPtrGetter func(KeyValueStorage, string) *int
		intPtrSetter func(KeyValueStorage, string, *int)
	)
	MakeKeyValueFuncs(&intPtrGetter, &intPtrSetter)
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
