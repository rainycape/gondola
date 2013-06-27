package orm

import (
	"fmt"
	"gondola/log"
	"gondola/orm/codec"
	"gondola/orm/driver"
	"gondola/orm/tag"
	"gondola/util"
	"reflect"
	"strings"
)

type nameRegistry map[string]*model
type typeRegistry map[reflect.Type]*model

var (
	// these keep track of the registered models,
	// using the driver tags as the key.
	_nameRegistry = map[string]nameRegistry{}
	_typeRegistry = map[string]typeRegistry{}
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
	var name string
	if opt != nil {
		name = opt.TableName
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
	model := &model{
		typ:       typ,
		fields:    fields,
		options:   opt,
		tableName: name,
		tags:      o.tags,
	}
	_nameRegistry[o.tags][name] = model
	// The first registered table is the default for the type
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

// CommitTables initializes any tables and indexes required by
// the registered models. You should call it only after all the
// models have been registered.
func (o *Orm) CommitTables() error {
	nr := _nameRegistry[o.tags]
	models := make([]driver.Model, 0, len(nr))
	for _, v := range nr {
		models = append(models, v)
	}
	return o.driver.MakeTables(models)
}

// MustCommitTables works like CommitTables, but panics if
// there's an error.
func (o *Orm) MustCommitTables() {
	if err := o.CommitTables(); err != nil {
		panic(err)
	}
}

func (o *Orm) fields(typ reflect.Type) (*driver.Fields, error) {
	methods, err := driver.MakeMethods(typ)
	if err != nil {
		return nil, err
	}
	fields := &driver.Fields{
		NameMap:    make(map[string]int),
		QNameMap:   make(map[string]int),
		PrimaryKey: -1,
		Methods:    methods,
	}
	if err := o._fields(typ, fields, "", "", nil); err != nil {
		return nil, err
	}
	return fields, nil
}

func (o *Orm) _fields(typ reflect.Type, fields *driver.Fields, prefix, dbPrefix string, index []int) error {
	n := typ.NumField()
	dtags := o.driver.Tags()
	for ii := 0; ii < n; ii++ {
		field := typ.Field(ii)
		if field.PkgPath != "" {
			// Unexported
			continue
		}
		ftag := tag.New(field, dtags)
		name := ftag.Name()
		if name == "-" {
			// Ignored field
			continue
		}
		if name == "" {
			// Default name
			name = util.CamelCaseToLower(field.Name, "_")
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
		// Check encoded types
		if c := codec.FromTag(ftag); c != nil {
			if err := c.Try(t, dtags); err != nil {
				return fmt.Errorf("can't encode field %q as %s: %s", qname, c.Name(), err)
			}
		} else if ftag.CodecName() != "" {
			return fmt.Errorf("can't find ORM codec %q. Perhaps you missed an import?", ftag.CodecName())
		} else {
			switch k {
			case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map:
				return fmt.Errorf("field %q in struct %v has invalid type %v", field.Name, typ, k)
			case reflect.Struct:
				// Inner struct
				idx := make([]int, len(index))
				copy(idx, index)
				idx = append(idx, field.Index[0])
				if t.Name() != "Time" || t.PkgPath() != "time" {
					prefix := dbPrefix
					if !ftag.Has("inline") {
						prefix += name + "_"
					}
					err := o._fields(t, fields, qname+".", prefix, idx)
					if err != nil {
						return err
					}
					continue
				}
			}
		}
		idx := make([]int, len(index))
		copy(idx, index)
		idx = append(idx, field.Index[0])
		fields.Names = append(fields.Names, name)
		fields.QNames = append(fields.QNames, qname)
		fields.OmitZero = append(fields.OmitZero, ftag.Has("omitzero") || (ftag.Has("auto_increment") && !ftag.Has("notomitzero")))
		fields.NullZero = append(fields.NullZero, ftag.Has("nullzero") || (defaultsToNullZero(field.Type.Kind(), ftag) && !ftag.Has("notnullzero")))
		fields.Indexes = append(fields.Indexes, idx)
		fields.Tags = append(fields.Tags, ftag)
		fields.Types = append(fields.Types, t)
		p := len(fields.Names) - 1
		fields.NameMap[name] = p
		fields.QNameMap[qname] = p
		if ftag.Has("primary_key") {
			if fields.PrimaryKey >= 0 {
				return fmt.Errorf("duplicate primary_key in struct %v (%s and %s)", typ, fields.Names[fields.PrimaryKey], name)
			}
			fields.PrimaryKey = len(fields.Names) - 1
			if ftag.Has("auto_increment") && (k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64) {
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
	return util.CamelCaseToLower(n, "_")
}

// returns wheter the kind defaults to nullzero option
func defaultsToNullZero(k reflect.Kind, t *tag.Tag) bool {
	return k == reflect.Slice || k == reflect.Ptr || k == reflect.Interface
}
