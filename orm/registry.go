package orm

import (
	"fmt"
	"gondola/log"
	"gondola/orm/driver"
	"gondola/util"
	"reflect"
	"strings"
)

type nameRegistry map[string]*Model
type typeRegistry map[reflect.Type]*Model

var (
	// these keep track of the registered models,
	// using the driver tags as the key.
	_nameRegistry = map[string]nameRegistry{}
	_typeRegistry = map[string]typeRegistry{}
)

// Register registers a struct for future usage with the ORMs with
// the same driver. If you're using ORM instances with different drivers
// (e.g. postgres and mongodb)  you must register each model with each
// driver (by creating an ORM of each type, calling Register() and then
// CommitModels(). The first returned value is a Model object, which must be
// using when querying the ORM.
func (o *Orm) Register(t interface{}, opt *Options) (*Model, error) {
	var name string
	if opt != nil {
		name = opt.Name
	}
	typ := reflect.TypeOf(t)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs can be registered as models (tried to register %T)", t)
	}
	if typ.NumField() == 0 {
		return nil, fmt.Errorf("type %T has no fields", t)
	}
	if name == "" {
		name = o.name(typ)
	}
	if _nameRegistry[o.tags] == nil {
		_nameRegistry[o.tags] = nameRegistry{}
		_typeRegistry[o.tags] = typeRegistry{}
	}
	if _, ok := _nameRegistry[o.tags][name]; ok {
		return nil, fmt.Errorf("duplicate model name %q", name)
	}
	fields, err := o.fields(typ)
	if err != nil {
		return nil, err
	}
	model := &Model{
		typ:        typ,
		fields:     fields,
		options:    opt,
		collection: name,
		tags:       o.tags,
	}
	_nameRegistry[o.tags][name] = model
	_typeRegistry[o.tags][typ] = model
	log.Debugf("Registered model %v (%q) with tags %q", typ, name, o.tags)
	return model, nil
}

// MustRegister works like Register, but panics if there's an
// error.
func (o *Orm) MustRegister(t interface{}, opt *Options) *Model {
	m, err := o.Register(t, opt)
	if err != nil {
		panic(err)
	}
	return m
}

func (o *Orm) CommitModels() error {
	nr := _nameRegistry[o.tags]
	models := make([]driver.Model, 0, len(nr))
	for _, v := range nr {
		models = append(models, v)
	}
	return o.driver.MakeModels(models)
}

// MustCommitModels works like CommitModels, but panics if
// there's an error.
func (o *Orm) MustCommitModels() {
	if err := o.CommitModels(); err != nil {
		panic(err)
	}
}

func (o *Orm) fields(typ reflect.Type) (*driver.Fields, error) {
	fields := &driver.Fields{
		NameMap:    make(map[string]int),
		QNameMap:   make(map[string]int),
		PrimaryKey: -1,
	}
	if err := o._fields(typ, fields, "", "", nil); err != nil {
		return nil, err
	}
	return fields, nil
}

func (o *Orm) _fields(typ reflect.Type, fields *driver.Fields, prefix, dbPrefix string, index []int) error {
	n := typ.NumField()
	for ii := 0; ii < n; ii++ {
		field := typ.Field(ii)
		if field.PkgPath != "" {
			// Unexported
			continue
		}
		tag := driver.NewTag(field, o.driver)
		name := tag.Name()
		if name == "-" {
			// Ignored field
			continue
		}
		if name == "" {
			// Default name
			name = util.UnCamelCase(field.Name, "_")
		}
		name = dbPrefix + name
		if _, ok := fields.NameMap[name]; ok {
			return fmt.Errorf("duplicate field %q in struct %v", name, typ)
		}
		qname := prefix + field.Name
		// Check type
		t := field.Type
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		k := t.Kind()
		switch k {
		case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map:
			return fmt.Errorf("field %q in struct %v has invalid type %v", field.Name, typ, k)
		case reflect.Struct:
			// Inner struct
			idx := make([]int, len(index))
			copy(idx, index)
			idx = append(idx, field.Index[0])
			if t.Name() != "Time" || t.PkgPath() != "time" {
				err := o._fields(t, fields, qname+".", dbPrefix+name+"_", idx)
				if err != nil {
					return err
				}
				continue
			}
		}
		idx := make([]int, len(index))
		copy(idx, index)
		idx = append(idx, field.Index[0])
		dt := driver.Tag(tag)
		fields.Names = append(fields.Names, name)
		fields.QNames = append(fields.QNames, qname)
		fields.OmitZero = append(fields.OmitZero, dt.Has("omitzero") || (dt.Has("auto_increment") && !dt.Has("notomitzero")))
		fields.NullZero = append(fields.NullZero, dt.Has("nullzero") || (k == reflect.Slice && !dt.Has("notnullzero")))
		fields.Indexes = append(fields.Indexes, idx)
		fields.Tags = append(fields.Tags, &dt)
		fields.Types = append(fields.Types, t)
		p := len(fields.Names) - 1
		fields.NameMap[name] = p
		fields.QNameMap[qname] = p
		if dt.Has("primary_key") {
			if fields.PrimaryKey >= 0 {
				return fmt.Errorf("duplicate primary_key in struct %v (%s and %s)", typ, fields.Names[fields.PrimaryKey], name)
			}
			fields.PrimaryKey = len(fields.Names) - 1
			if dt.Has("auto_increment") && (k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64) {
				fields.IntegerAutoincrementPk = true
			}
		}
	}
	return nil
}

func (o *Orm) name(typ reflect.Type) string {
	n := typ.Name()
	if p := typ.PkgPath(); p != "main" {
		n = strings.Replace(p, "/", "_", -1) + n
	}
	return util.UnCamelCase(n, "_")
}
