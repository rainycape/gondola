package cache

import (
	"reflect"
)

// Typer returns the type of object to be decoded for
// a given key.
type Typer interface {
	Type(key string) reflect.Type
}

type uniTyper struct {
	typ reflect.Type
}

func (t *uniTyper) Type(_ string) reflect.Type {
	return t.typ
}

// UniType returns a Typer which returns the type of
// obj for all the keys.
func UniTyper(obj interface{}) Typer {
	return &uniTyper{typ: reflect.TypeOf(obj)}
}

type mapTyper map[string]interface{}

func (t mapTyper) Type(key string) reflect.Type {
	return reflect.TypeOf(t[key])
}
