package types

import (
	"reflect"
	"strings"
)

type Tag struct {
	name   string
	values map[string]string
}

func (t *Tag) Name() string {
	return t.name
}

func (t *Tag) Has(opt string) bool {
	_, ok := t.values[opt]
	return ok
}

func (t *Tag) Value(key string) string {
	return t.values[key]
}

func (t *Tag) CodecName() string {
	return t.Value("codec")
}

func (t *Tag) IsEmpty() bool {
	return t.name == "" && len(t.values) == 0
}

func makeTag(tag string) *Tag {
	fields := strings.Split(tag, ",")
	name := fields[0]
	values := make(map[string]string, len(fields)-1)
	for _, v := range fields[1:] {
		idx := strings.Index(v, ":")
		if idx >= 0 {
			values[v[:idx]] = v[idx+1:]
		} else {
			values[v] = ""
		}
	}
	return &Tag{name, values}
}

func NewTag(field reflect.StructField, alternatives []string) *Tag {
	for _, v := range alternatives {
		t := field.Tag.Get(v)
		if t != "" {
			return makeTag(t)
		}
	}
	return makeTag("")
}

func NewTagNamed(field reflect.StructField, name string) *Tag {
	return makeTag(field.Tag.Get(name))
}
