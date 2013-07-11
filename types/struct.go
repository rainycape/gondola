package types

import (
	"errors"
	"fmt"
	"gondola/util"
	"reflect"
)

var (
	ErrNoStruct = errors.New("not an struct")
	ErrNoFields = errors.New("struct has no fields")
)

type Struct struct {
	// The Struct type
	Type reflect.Type
	// Lists the mangled names of the fields, in order
	MNames []string
	// List the names of the qualified struct fields (e.g. Foo.Bar) in order
	QNames []string
	// Lists the indexes of the members (for FieldByIndex())
	Indexes [][]int
	// Field types, in order
	Types []reflect.Type
	// Field tags, in order
	Tags []*Tag
	// Maps mangled names to indexes
	MNameMap map[string]int
	// Maps qualified names to indexes
	QNameMap map[string]int
}

// Map takes a qualified struct name and returns its mangled name and type
func (s *Struct) Map(qname string) (string, reflect.Type, error) {
	if n, ok := s.QNameMap[qname]; ok {
		return s.MNames[n], s.Types[n], nil
	}
	return "", nil, fmt.Errorf("can't map field %q to a mangled name", qname)
}

func NewStruct(t interface{}, tags []string) (*Struct, error) {
	typ := reflect.TypeOf(t)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, ErrNoStruct
	}
	if typ.NumField() == 0 {
		return nil, ErrNoFields
	}
	s := &Struct{
		Type:     typ,
		MNameMap: make(map[string]int),
		QNameMap: make(map[string]int),
	}
	if err := fields(typ, tags, s, "", "", nil); err != nil {
		return nil, err
	}
	return s, nil
}

func fields(typ reflect.Type, tags []string, s *Struct, qprefix, mprefix string, index []int) error {
	n := typ.NumField()
	for ii := 0; ii < n; ii++ {
		field := typ.Field(ii)
		if field.PkgPath != "" {
			// Unexported
			continue
		}
		ftag := NewTag(field, tags)
		name := ftag.Name()
		if name == "-" {
			// Ignored field
			continue
		}
		if name == "" {
			// Default name
			name = util.CamelCaseToLower(field.Name, "_")
		}
		name = mprefix + name
		if _, ok := s.MNameMap[name]; ok {
			return fmt.Errorf("duplicate field %q in struct %v", name, typ)
		}
		qname := qprefix + field.Name
		// Check type
		t := field.Type
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		// TODO: The ORM needs the fields tagged with a codec
		// to not be broken into their members. Make this a
		// parameter, since other users of this function
		// might want all the fields.
		if t.Kind() == reflect.Struct && !ftag.Has("codec") {
			// Inner struct
			idx := make([]int, len(index))
			copy(idx, index)
			idx = append(idx, field.Index[0])
			if t.Name() != "Time" || t.PkgPath() != "time" {
				prefix := mprefix
				if !ftag.Has("inline") {
					prefix += name + "_"
				}
				err := fields(t, tags, s, qname+".", prefix, idx)
				if err != nil {
					return err
				}
				continue
			}
		}
		idx := make([]int, len(index))
		copy(idx, index)
		idx = append(idx, field.Index[0])
		s.MNames = append(s.MNames, name)
		s.QNames = append(s.QNames, qname)
		s.Indexes = append(s.Indexes, idx)
		s.Tags = append(s.Tags, ftag)
		s.Types = append(s.Types, t)
		p := len(s.MNames) - 1
		s.MNameMap[name] = p
		s.QNameMap[qname] = p
	}
	return nil
}
