package orm

import (
	"fmt"
	"gondola/log"
	"gondola/orm/driver"
	"gondola/util"
	"reflect"
	"strings"
)

var (
	nameRegistry = map[string]*Model{}
	typeRegistry = map[reflect.Type]*Model{}
)

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
	if _, ok := nameRegistry[name]; ok {
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
	}
	nameRegistry[name] = model
	typeRegistry[typ] = model
	log.Infof("Registered model %v (%q) %+v, fields %+v", typ, name, model, fields)
	return model, nil
}

func (o *Orm) MustRegister(t interface{}, opt *Options) *Model {
	m, err := o.Register(t, opt)
	if err != nil {
		panic(err)
	}
	return m
}

func (o *Orm) CommitModels() error {
	models := make([]driver.Model, 0, len(nameRegistry))
	for _, v := range nameRegistry {
		models = append(models, v)
	}
	return o.driver.MakeModels(models)
}

func (o *Orm) MustCommitModels() {
	if err := o.CommitModels(); err != nil {
		panic(err)
	}
}

func (o *Orm) fields(typ reflect.Type) (*driver.Fields, error) {
	fields := &driver.Fields{
		Types:      make(map[string]reflect.Type),
		Tags:       make(map[string]driver.Tag),
		NameMap:    make(map[string]string),
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
		tag := field.Tag.Get("orm")
		if tag == "" {
			if t := o.driver.Tag(); t != "" {
				tag = field.Tag.Get(t)
			}
		}
		var name string
		if tag != "" {
			t := driver.Tag(tag)
			if n := t.Name(); n != "" {
				if n == "-" {
					// Ignored field
					continue
				}
				name = n
			}
		}
		if name == "" {
			// Default name
			name = util.UnCamelCase(field.Name, "_")
		}
		name = dbPrefix + name
		if _, ok := fields.Tags[name]; ok {
			return fmt.Errorf("duplicate field %q in struct %v", name, typ)
		}
		// Check type
		t := field.Type
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		k := t.Kind()
		switch k {
		case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice:
			return fmt.Errorf("field %q in struct %v has invalid type %v", field.Name, typ, k)
		case reflect.Struct:
			// Inner struct
			idx := make([]int, len(index))
			copy(idx, index)
			idx = append(idx, field.Index[0])
			if t.Name() != "Time" || t.PkgPath() != "time" {
				err := o._fields(t, fields, prefix+field.Name+".", dbPrefix+name+"_", idx)
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
		fields.OmitZero = append(fields.OmitZero, dt.Has("omitzero") || (dt.Has("auto_increment") && !dt.Has("notomitzero")))
		fields.NullZero = append(fields.NullZero, dt.Has("nullzero"))
		fields.Indexes = append(fields.Indexes, idx)
		fields.Tags[name] = dt
		fields.Types[name] = t
		fields.NameMap[prefix+field.Name] = name
		if dt.Has("primary_key") {
			if fields.PrimaryKey >= 0 {
				return fmt.Errorf("duplicate primary_key in struct %v (%s and %s)", typ, fields.Names[fields.PrimaryKey], name)
			}
			fields.PrimaryKey = len(fields.Names) - 1
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
