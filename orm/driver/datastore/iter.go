// +build appengine

package datastore

import (
	"reflect"

	"appengine/datastore"
)

type Iter struct {
	iter *datastore.Iterator
	err  error
}

func (i *Iter) Next(out ...interface{}) bool {
	dst := out[0]
	val := reflect.ValueOf(dst)
	// Pointer to pointer, must pass a simple pointer to iter.Next()
	var dstVal reflect.Value
	if val.IsValid() && val.Kind() == reflect.Ptr && val.Type().Elem().Kind() == reflect.Ptr {
		dstVal = reflect.New(val.Type().Elem().Elem())
		dst = dstVal.Interface()
	}
	if i.err == nil {
		_, i.err = i.iter.Next(dst)
		if i.err == nil && dstVal.IsValid() {
			val.Elem().Set(dstVal)
		}
	}
	return i.err == nil
}

func (i *Iter) Err() error {
	if i.err == datastore.Done {
		return nil
	}
	return i.err
}

func (i *Iter) Close() error {
	return nil
}
