package orm

import (
	"fmt"
	"gnd.la/log"
	"gnd.la/orm/codec"
	"gnd.la/orm/driver"
	"gnd.la/types"
	"gnd.la/util"
	"reflect"
	"strings"
	"time"
)

type nameRegistry map[string]*model
type typeRegistry map[reflect.Type]*model

var (
	// these keep track of the registered models,
	// using the driver tags as the key.
	_nameRegistry = map[string]nameRegistry{}
	_typeRegistry = map[string]typeRegistry{}
	timeType      = reflect.TypeOf(time.Time{})
)

// Register registers a struct for future usage with the ORMs with
// the same driver. If you're using ORM instances with different drivers
// (e.g. postgres and mongodb)  you must register each object type with each
// driver (by creating an ORM of each type, calling Register() and then
// CommitTables()). The first returned value is a Table object, which must be
// using when querying the ORM in cases when an object is not provided
// (like e.g. Count()). If you want to use the same type in multiple
// tables, you must register it for every table and then use the Table
// object returned to specify on which table you want to operate. If
// no table is specified, the first registered table will be used.
func (o *Orm) Register(t interface{}, opt *Options) (*Table, error) {
	s, err := types.NewStruct(t, o.dtags())
	if err != nil {
		switch err {
		case types.ErrNoStruct:
			return nil, fmt.Errorf("only structs can be registered as models (tried to register %T)", t)
		case types.ErrNoFields:
			return nil, fmt.Errorf("type %T has no fields", t)
		}
		return nil, err
	}
	var name string
	if opt != nil {
		name = opt.Table
	}
	if name == "" {
		name = defaultName(s.Type)
	}
	if _nameRegistry[o.tags] == nil {
		_nameRegistry[o.tags] = nameRegistry{}
		_typeRegistry[o.tags] = typeRegistry{}
	}
	if _, ok := _nameRegistry[o.tags][name]; ok {
		return nil, fmt.Errorf("duplicate ORM model name %q", name)
	}
	fields, err := o.fields(s)
	if err != nil {
		return nil, err
	}
	if opt != nil {
		if len(opt.PrimaryKey) > 0 {
			if fields.PrimaryKey >= 0 {
				return nil, fmt.Errorf("duplicate primary key in model %q. tags define %q as PK, options define %v",
					name, fields.QNames[fields.PrimaryKey], opt.PrimaryKey)
			}
			fields.CompositePrimaryKey = make([]int, len(opt.PrimaryKey))
			for ii, v := range opt.PrimaryKey {
				pos, ok := fields.QNameMap[v]
				if !ok {
					return nil, fmt.Errorf("can't map qualified name %q on model %q when creating composite key", v, name)
				}
				fields.CompositePrimaryKey[ii] = pos
			}
		}
	}
	model := &model{
		fields:  fields,
		options: opt,
		table:   name,
		tags:    o.tags,
	}
	_nameRegistry[o.tags][name] = model
	// The first registered table is the default for the type
	typ := model.Type()
	if _, ok := _typeRegistry[o.tags][typ]; !ok {
		_typeRegistry[o.tags][typ] = model
	}
	log.Debugf("Registered model %v (%q) with tags %q", typ, name, o.tags)
	return &Table{model: model}, nil
}

// MustRegister works like Register, but panics if there's an
// error.
func (o *Orm) MustRegister(t interface{}, opt *Options) *Table {
	tbl, err := o.Register(t, opt)
	if err != nil {
		panic(err)
	}
	return tbl
}

// Initialize initializes any tables and indexes required by
// the registered models. You MUST call it AFTER all the
// models have been registered and BEFORE starting to use the ORM
// from several goroutines (for performance reasons, the access
// to some shared resources from several ORM instances is not
// thread safe).
func (o *Orm) Initialize() error {
	nr := _nameRegistry[o.tags]
	models := make([]driver.Model, 0, len(nr))
	for _, v := range nr {
		models = append(models, v)
	}
	return o.driver.MakeTables(models)
}

// MustInitialize works like Initialize, but panics if
// there's an error.
func (o *Orm) MustInitialize() {
	if err := o.Initialize(); err != nil {
		panic(err)
	}
}

func (o *Orm) fields(s *types.Struct) (*driver.Fields, error) {
	methods, err := driver.MakeMethods(s.Type)
	if err != nil {
		return nil, err
	}
	fields := &driver.Fields{
		Struct:     s,
		PrimaryKey: -1,
		Methods:    methods,
	}
	for ii, v := range s.QNames {
		t := s.Types[ii]
		k := t.Kind()
		ftag := s.Tags[ii]
		// Check encoded types
		if c := codec.FromTag(ftag); c != nil {
			if err := c.Try(t, o.dtags()); err != nil {
				return nil, fmt.Errorf("can't encode field %q as %s: %s", v, c.Name(), err)
			}
		} else if ftag.CodecName() != "" {
			return nil, fmt.Errorf("can't find ORM codec %q. Perhaps you missed an import?", ftag.CodecName())
		} else {
			switch k {
			case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map:
				return nil, fmt.Errorf("field %q in struct %v has invalid type %v", v, t, k)
			}
		}
		fields.OmitEmpty = append(fields.OmitEmpty, ftag.Has("omitempty") || (ftag.Has("auto_increment") && !ftag.Has("notomitempty")))
		// Struct has flattened types, but we need to original type
		// to determine if it should be nullempty by default
		field := s.Type.FieldByIndex(s.Indexes[ii])
		fields.NullEmpty = append(fields.NullEmpty, ftag.Has("nullempty") || (defaultsToNullEmpty(field.Type, ftag) && !ftag.Has("notnullempty")))
		if ftag.Has("primary_key") {
			if fields.PrimaryKey >= 0 {
				return nil, fmt.Errorf("duplicate primary_key in struct %v (%s and %s)", s.Type, s.QNames[fields.PrimaryKey], v)
			}
			fields.PrimaryKey = ii
			if ftag.Has("auto_increment") && (k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64) {
				fields.IntegerAutoincrementPk = true
			}
		}
	}
	return fields, nil
}

func (o *Orm) dtags() []string {
	return append(o.driver.Tags(), "orm")
}

// returns wheter the kind defaults to nullempty option
func defaultsToNullEmpty(typ reflect.Type, t *types.Tag) bool {
	switch typ.Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Interface, reflect.String:
		return true
	case reflect.Struct:
		return typ == timeType
	}
	return false
}

// Returns the default name for a type
func defaultName(typ reflect.Type) string {
	n := typ.Name()
	if p := typ.PkgPath(); p != "main" {
		n = strings.Replace(p, "/", "_", -1) + n
	}
	return util.CamelCaseToLower(n, "_")
}
