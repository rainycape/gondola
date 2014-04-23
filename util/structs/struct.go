package structs

import (
	"errors"
	"fmt"
	"gnd.la/util/stringutil"
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
	// Lists the field indexes prefix for pointers in embedded structs
	Pointers [][]int
}

// Map takes a qualified struct name and returns its mangled name and type
func (s *Struct) Map(qname string) (string, reflect.Type, error) {
	if n, ok := s.QNameMap[qname]; ok {
		return s.MNames[n], s.Types[n], nil
	}
	return "", nil, fmt.Errorf("can't map field %q to a mangled name", qname)
}

// Embeds returns true iff the struct embeds the given type.
func (s *Struct) Embeds(typ reflect.Type) bool {
	end := s.Type.NumField()
	for ii := 0; ii < end; ii++ {
		field := s.Type.Field(ii)
		if field.Type == typ && field.Anonymous {
			return true
		}
	}
	return false
}

func NewStruct(t interface{}, tags []string) (*Struct, error) {
	var typ reflect.Type
	if tt, ok := t.(reflect.Type); ok {
		typ = tt
	} else {
		typ = reflect.TypeOf(t)
	}
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
			name = stringutil.CamelCaseToLower(field.Name, "_")
		}
		name = mprefix + name
		if _, ok := s.MNameMap[name]; ok {
			return fmt.Errorf("duplicate field %q in struct %v", name, typ)
		}
		qname := qprefix + field.Name
		// Check type
		ptr := false
		t := field.Type
		for t.Kind() == reflect.Ptr {
			ptr = true
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct && decompose(t, ftag) {
			// Inner struct
			idx := make([]int, len(index))
			copy(idx, index)
			idx = append(idx, field.Index[0])
			prefix := mprefix
			if !ftag.Has("inline") {
				prefix += name + "_"
			}
			err := fields(t, tags, s, qname+".", prefix, idx)
			if err != nil {
				return err
			}
			if ptr {
				s.Pointers = append(s.Pointers, idx)
			}
			continue
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

// Returns wheter a stuct should decomposed into its fields
func decompose(typ reflect.Type, tag *Tag) bool {
	// TODO: The ORM needs the fields tagged with a codec
	// to not be broken into their members. Make this a
	// parameter, since other users of this function
	// might want all the fields. Make also struct types
	// like time.Time configurable
	return !tag.Has("codec") && !(typ.Name() == "Time" && typ.PkgPath() == "time")
}
