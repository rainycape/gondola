package structs

import (
	"errors"
	"fmt"
	"reflect"

	"gnd.la/util/stringutil"
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
	tags     []string
	conf     Configurator
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
	return s.Has("", typ)
}

// Has returns true iff the struct has a field with the given name
// of the given type. If field is empty, it works like Embeds.
func (s *Struct) Has(field string, typ reflect.Type) bool {
	end := s.Type.NumField()
	for ii := 0; ii < end; ii++ {
		f := s.Type.Field(ii)
		if f.Type == typ && ((field == "" && f.Anonymous) || field == f.Name) {
			return true
		}
	}
	return false
}

func (s *Struct) initialize(typ reflect.Type) error {
	if err := s.initializeFields(typ, "", "", nil); err != nil {
		return err
	}
	return nil
}

func (s *Struct) initializeFields(typ reflect.Type, qprefix, mprefix string, index []int) error {
	n := typ.NumField()
	for ii := 0; ii < n; ii++ {
		field := typ.Field(ii)
		if field.PkgPath != "" {
			// Unexported
			continue
		}
		ftag := NewTag(field, s.tags)
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
		qname := qprefix + field.Name
		if prev, ok := s.MNameMap[name]; ok {
			return fmt.Errorf("duplicate field %q in %s: %s and %s", name, s.Type, s.QNames[prev], qname)
		}
		// Check type
		ptr := false
		t := field.Type
		for t.Kind() == reflect.Ptr {
			ptr = true
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct && s.decomposeField(t, ftag) {
			// Inner struct
			idx := make([]int, len(index))
			copy(idx, index)
			idx = append(idx, field.Index[0])
			prefix := mprefix
			if !ftag.Has("inline") {
				prefix += name + "_"
			}
			err := s.initializeFields(t, qname+".", prefix, idx)
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

func (s *Struct) decomposeField(typ reflect.Type, tag *Tag) bool {
	if s.conf != nil {
		return s.conf.DecomposeField(s, typ, tag)
	}
	return true
}

type Configurator interface {
	// Returns wheter a Struct should decompose the given struct field
	// into its fields or just use the struct as is.
	DecomposeField(s *Struct, typ reflect.Type, tag *Tag) bool
}

func New(t interface{}, tags []string, conf Configurator) (*Struct, error) {
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
		tags:     tags,
		conf:     conf,
	}
	if err := s.initialize(typ); err != nil {
		return nil, err
	}
	return s, nil
}
