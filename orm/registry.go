package orm

import (
	"fmt"
	"gnd.la/log"
	"gnd.la/orm/codec"
	"gnd.la/orm/driver"
	"gnd.la/orm/query"
	"gnd.la/types"
	"gnd.la/util"
	"reflect"
	"regexp"
	"sort"
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
	referencesRe  = regexp.MustCompile("([\\w\\.]+)(\\((\\w+)\\))?")
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
	var table string
	if opt != nil {
		table = opt.Table
	}
	if table == "" {
		table = defaultTableName(s.Type)
	}
	if _nameRegistry[o.tags] == nil {
		_nameRegistry[o.tags] = nameRegistry{}
		_typeRegistry[o.tags] = typeRegistry{}
	}
	if _, ok := _nameRegistry[o.tags][table]; ok {
		return nil, fmt.Errorf("duplicate ORM table name %q", table)
	}
	fields, references, err := o.fields(table, s)
	if err != nil {
		return nil, err
	}
	var name string
	if opt != nil && opt.Name != "" {
		name = opt.Name
	} else {
		name = typeName(s.Type)
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
		fields:     fields,
		name:       name,
		shortName:  s.Type.Name(),
		references: references,
		options:    opt,
		table:      table,
		tags:       o.tags,
	}
	_nameRegistry[o.tags][table] = model
	// The first registered table is the default for the type
	typ := model.Type()
	if _, ok := _typeRegistry[o.tags][typ]; !ok || opt != nil && opt.Default {
		_typeRegistry[o.tags][typ] = model
	}
	log.Debugf("Registered model %v (%q) with tags %q", typ, name, o.tags)
	return &Table{model: &joinModel{model: model}}, nil
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

// Initialize resolves model references and creates tables and
// indexes required by the registered models. You MUST call it
// AFTER all the models have been registered and BEFORE starting
// to use the ORM from several goroutines (for performance reasons,
// the access to some shared resources from several ORM instances
// is not thread safe).
func (o *Orm) Initialize() error {
	nr := _nameRegistry[o.tags]
	// Resolve references
	names := make(map[string]*model)
	for _, v := range nr {
		names[v.name] = v
	}
	for _, v := range nr {
		if c := len(v.references); c > 0 {
			v.fields.References = make(map[string]*driver.Reference, c)
			for k, r := range v.references {
				referenced := names[r.model]
				if referenced == nil {
					if !strings.Contains(r.model, ".") {
						referenced = names[v.Type().PkgPath()+"."+r.model]
					}
					if referenced == nil {
						return fmt.Errorf("can't find referenced model %q from model %q", r.model, v.name)
					}
				}
				if r.field == "" {
					// Map to PK
					if pk := referenced.fields.PrimaryKey; pk >= 0 {
						r.field = referenced.fields.QNames[pk]
					} else {
						return fmt.Errorf("referenced model %q does not have a non-composite primary key. Please, specify a field", r.model)
					}
				}
				_, ft, err := v.fields.Map(k)
				if err != nil {
					return err
				}
				_, fkt, err := referenced.fields.Map(r.field)
				if err != nil {
					return err
				}
				if ft != fkt {
					return fmt.Errorf("type mismatch: referenced field %q in model %q is of type %s, field %q in model %q is of type %s",
						r.field, referenced.name, fkt, k, v.name, ft)
				}
				v.fields.References[k] = &driver.Reference{
					Model: referenced,
					Field: r.field,
				}
				if v.modelReferences == nil {
					v.modelReferences = make(map[*model][]*join)
				}
				v.modelReferences[referenced] = append(v.modelReferences[referenced], &join{
					model: &joinModel{model: referenced},
					q:     Eq(v.fullName(k), query.F(referenced.fullName(r.field))),
				})
				if referenced.modelReferences == nil {
					referenced.modelReferences = make(map[*model][]*join)
				}
				referenced.modelReferences[v] = append(referenced.modelReferences[v], &join{
					model: &joinModel{model: v},
					q:     Eq(referenced.fullName(r.field), query.F(v.fullName(k))),
				})
			}
		}
	}
	models := make([]driver.Model, 0, len(nr))
	for _, v := range nr {
		models = append(models, v)
	}
	// Sort models to the ones with FKs are created after
	// the models they reference
	sort.Sort(sortModels(models))
	return o.driver.MakeTables(models)
}

// MustInitialize works like Initialize, but panics if
// there's an error.
func (o *Orm) MustInitialize() {
	if err := o.Initialize(); err != nil {
		panic(err)
	}
}

func (o *Orm) fields(table string, s *types.Struct) (*driver.Fields, map[string]*reference, error) {
	methods, err := driver.MakeMethods(s.Type)
	if err != nil {
		return nil, nil, err
	}
	fields := &driver.Fields{
		Struct:     s,
		PrimaryKey: -1,
		Methods:    methods,
	}
	var references map[string]*reference
	for ii, v := range s.QNames {
		// XXX: Check if this quoting is enough
		fields.QuotedNames = append(fields.QuotedNames, fmt.Sprintf("\"%s\".\"%s\"", table, s.MNames[ii]))
		t := s.Types[ii]
		k := t.Kind()
		ftag := s.Tags[ii]
		// Check encoded types
		if c := codec.FromTag(ftag); c != nil {
			if err := c.Try(t, o.dtags()); err != nil {
				return nil, nil, fmt.Errorf("can't encode field %q as %s: %s", v, c.Name(), err)
			}
		} else if ftag.CodecName() != "" {
			return nil, nil, fmt.Errorf("can't find ORM codec %q. Perhaps you missed an import?", ftag.CodecName())
		} else {
			switch k {
			case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map:
				return nil, nil, fmt.Errorf("field %q in struct %v has invalid type %v", v, t, k)
			}
		}
		fields.OmitEmpty = append(fields.OmitEmpty, ftag.Has("omitempty") || (ftag.Has("auto_increment") && !ftag.Has("notomitempty")))
		// Struct has flattened types, but we need to original type
		// to determine if it should be nullempty by default
		field := s.Type.FieldByIndex(s.Indexes[ii])
		fields.NullEmpty = append(fields.NullEmpty, ftag.Has("nullempty") || (defaultsToNullEmpty(field.Type, ftag) && !ftag.Has("notnullempty")))
		if ftag.Has("primary_key") {
			if fields.PrimaryKey >= 0 {
				return nil, nil, fmt.Errorf("duplicate primary_key in struct %v (%s and %s)", s.Type, s.QNames[fields.PrimaryKey], v)
			}
			fields.PrimaryKey = ii
			if ftag.Has("auto_increment") && (k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64) {
				fields.IntegerAutoincrementPk = true
			}
		}
		if ref := ftag.Value("references"); ref != "" {
			m := referencesRe.FindStringSubmatch(ref)
			if len(m) != 4 {
				return nil, nil, fmt.Errorf("field %q has invalid references %q. Must be in the form references=Model or references=Model(Field)", v, ref)
			}
			if references == nil {
				references = make(map[string]*reference)
			}
			references[v] = &reference{model: m[1], field: m[3]}
		}
	}
	return fields, references, nil
}

func (o *Orm) dtags() []string {
	return append(o.driver.Tags(), "orm")
}

// returns wheter the kind defaults to nullempty option
func defaultsToNullEmpty(typ reflect.Type, t *types.Tag) bool {
	if t.Has("references") || t.Has("codec") {
		return true
	}
	switch typ.Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Interface, reflect.String:
		return true
	case reflect.Struct:
		return typ == timeType
	}
	return false
}

// Returns the default table name for a type
func defaultTableName(typ reflect.Type) string {
	n := typ.Name()
	if p := typ.PkgPath(); p != "main" {
		n = strings.Replace(p, "/", "_", -1) + n
	}
	return util.CamelCaseToLower(n, "_")
}

func typeName(typ reflect.Type) string {
	if p := typ.PkgPath(); p != "main" {
		return p + "." + typ.Name()
	}
	return typ.Name()
}
