package orm

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gnd.la/encoding/codec"
	"gnd.la/encoding/pipe"
	"gnd.la/form/input"
	"gnd.la/log"
	"gnd.la/orm/driver"
	"gnd.la/orm/query"
	"gnd.la/signal"
	"gnd.la/util/stringutil"
	"gnd.la/util/structs"
	"gnd.la/util/types"
)

type nameRegistry map[string]*model
type typeRegistry map[reflect.Type]*model

func (r typeRegistry) clone() typeRegistry {
	cpy := make(typeRegistry, len(r))
	for k, v := range r {
		cpy[k] = v
	}
	return cpy
}

type pending struct {
	typ  reflect.Type
	opts *Options
}

var (
	timeType     = reflect.TypeOf(time.Time{})
	referencesRe = regexp.MustCompile("([\\w\\.]+)(\\((\\w+)\\))?")

	globalRegistry struct {
		sync.RWMutex
		// these keep track of the registered models,
		// using the driver tags as the key.
		names map[string]nameRegistry
		types map[string]typeRegistry
	}

	// models registered via orm.Register, need to be
	// added to all future Orm instances in Initialize.
	pendingRegistry struct {
		sync.RWMutex
		pending []*pending
	}
)

// Register registers a new type for all ORMs instantiated after
// this point. This is the preferred way to register structs and
// it generally should be called from an init() function.
func Register(t interface{}, opts *Options) {
	pendingRegistry.Lock()
	defer pendingRegistry.Unlock()
	var typ reflect.Type
	if tt, ok := t.(reflect.Type); ok {
		typ = tt
	} else {
		typ = reflect.TypeOf(t)
	}
	pendingRegistry.pending = append(pendingRegistry.pending, &pending{typ, opts})
}

// Register is considered a low-level function and should only be
// used if your app uses multiple ORM sources (e.g. Postgres and
// MongoDB). Most of the time, users should use orm.Register()
//
// Register registers a struct for future usage with the ORMs with
// the same driver. If you're using ORM instances with different drivers
// you must register each object type with each driver by creating an ORM
// of each type, calling Register() and then
// Initialize. The first returned value is a Table object, which must be
// using when querying the ORM in cases when an object is not provided
// (like e.g. Count()). If you want to use the same type in multiple
// tables, you must register it for every table and then use the Table
// object returned to specify on which table you want to operate. If
// no table is specified, the first registered table will be used.
func (o *Orm) Register(t interface{}, opts *Options) (*Table, error) {
	globalRegistry.Lock()
	defer globalRegistry.Unlock()
	return o.registerLocked(t, opts)
}

func (o *Orm) registerLocked(t interface{}, opts *Options) (*Table, error) {
	s, err := structs.NewStruct(t, o.dtags())
	if err != nil {
		switch err {
		case structs.ErrNoStruct:
			return nil, fmt.Errorf("only structs can be registered as models (tried to register %T)", t)
		case structs.ErrNoFields:
			return nil, fmt.Errorf("type %T has no fields", t)
		}
		return nil, err
	}
	var table string
	if opts != nil {
		table = opts.Table
	}
	if table == "" {
		table = defaultTableName(s.Type)
	}
	if globalRegistry.names[o.tags] == nil {
		globalRegistry.names[o.tags] = nameRegistry{}
		globalRegistry.types[o.tags] = typeRegistry{}
	}
	names := globalRegistry.names[o.tags]
	types := globalRegistry.types[o.tags]
	if _, ok := names[table]; ok {
		return nil, fmt.Errorf("duplicate ORM table name %q", table)
	}
	if _, ok := types[s.Type]; ok {
		return nil, fmt.Errorf("duplicate ORM type %s", s.Type)
	}
	fields, references, err := o.fields(table, s)
	if err != nil {
		return nil, err
	}
	var name string
	if opts != nil && opts.Name != "" {
		name = opts.Name
	} else {
		name = typeName(s.Type)
	}
	if opts != nil {
		if len(opts.PrimaryKey) > 0 {
			if fields.PrimaryKey >= 0 {
				return nil, fmt.Errorf("duplicate primary key in model %q. tags define %q as PK, Options define %v",
					name, fields.QNames[fields.PrimaryKey], opts.PrimaryKey)
			}
			fields.CompositePrimaryKey = make([]int, len(opts.PrimaryKey))
			for ii, v := range opts.PrimaryKey {
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
		options:    opts,
		table:      table,
		tags:       o.tags,
	}
	names[table] = model
	types[s.Type] = model
	log.Debugf("Registered model %v (%q) with tags %q", s.Type, name, o.tags)
	o.typeRegistry = types.clone()
	return tableWithModel(model), nil
}

func (o *Orm) initializePending() error {
	pendingRegistry.RLock()
	defer pendingRegistry.RUnlock()
	for _, v := range pendingRegistry.pending {
		if _, err := o.registerLocked(v.typ, v.opts); err != nil {
			return err
		}
	}
	return nil
}

// Initialize is a low level function and should only be used
// when dealing with multiple ORM types. If you're only using the
// default ORM as returned by gnd.la/app.App.Orm() or
// gnd.la/app.Context.Orm() you should not call this function
// manually.
//
// Initialize resolves model references and creates tables and
// indexes required by the registered models. You MUST call it
// AFTER all the models have been registered and BEFORE starting
// to use the ORM for queries for each ORM type.
func (o *Orm) Initialize() error {
	globalRegistry.Lock()
	defer globalRegistry.Unlock()
	signal.Emit(WILL_INITIALIZE, o)
	if err := o.initializePending(); err != nil {
		return err
	}
	nr := globalRegistry.names[o.tags]
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
				if v.namedReferences == nil {
					v.namedReferences = make(map[string]*model)
				}
				v.namedReferences[referenced.name] = referenced
				v.namedReferences[referenced.shortName] = referenced
				if referenced.modelReferences == nil {
					referenced.modelReferences = make(map[*model][]*join)
				}
				referenced.modelReferences[v] = append(referenced.modelReferences[v], &join{
					model: &joinModel{model: v},
					q:     Eq(referenced.fullName(r.field), query.F(v.fullName(k))),
				})
				if referenced.namedReferences == nil {
					referenced.namedReferences = make(map[string]*model)
				}
				referenced.namedReferences[v.name] = v
				referenced.namedReferences[v.shortName] = v
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
	return o.driver.Initialize(models)
}

func (o *Orm) fields(table string, s *structs.Struct) (*driver.Fields, map[string]*reference, error) {
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
		ftag := s.Tags[ii]
		// Check encoded types
		if cn := ftag.CodecName(); cn != "" {
			if codec.Get(cn) == nil {
				if imp := codec.RequiredImport(cn); imp != "" {
					return nil, nil, fmt.Errorf("please import %q to use the codec %q", imp, cn)
				}
				return nil, nil, fmt.Errorf("can't find codec %q. Perhaps you missed an import?", cn)
			}
		} else {
			switch t.Kind() {
			case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map:
				return nil, nil, fmt.Errorf("field %q in struct %s has invalid type %s", v, s.Type, t)
			}
		}
		if pn := ftag.PipeName(); pn != "" {
			// Check if the field has a codec and the pipe exists
			if ftag.CodecName() == "" {
				return nil, nil, fmt.Errorf("field %q has pipe %s but no codec - only encoded types can use pipes", v, pn)
			}
			if pipe.FromTag(ftag) == nil {
				return nil, nil, fmt.Errorf("can't find ORM pipe %q. Perhaps you missed an import?", pn)
			}
		}
		// Struct has flattened types, but we need to original type
		// to determine if it should be nullempty or omitempty by default
		field := s.Type.FieldByIndex(s.Indexes[ii])
		fields.OmitEmpty = append(fields.OmitEmpty, ftag.Has("omitempty") || (defaultsToOmitEmpty(field.Type, ftag) && !ftag.Has("notomitempty")))
		fields.NullEmpty = append(fields.NullEmpty, ftag.Has("nullempty") || (defaultsToNullEmpty(field.Type, ftag) && !ftag.Has("notnullempty")))
		if ftag.Has("primary_key") {
			if fields.PrimaryKey >= 0 {
				return nil, nil, fmt.Errorf("duplicate primary_key in struct %v (%s and %s)", s.Type, s.QNames[fields.PrimaryKey], v)
			}
			fields.PrimaryKey = ii
		}
		if ftag.Has("auto_increment") {
			if k := types.Kind(t.Kind()); k != types.Int && k != types.Uint {
				return nil, nil, fmt.Errorf("auto_increment field %q in struct %s must be of integer type (signed or unsigned", v, s.Type)
			}
			fields.AutoincrementPk = fields.PrimaryKey == ii
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
	if err := o.setFieldsDefaults(fields); err != nil {
		return nil, nil, err
	}
	return fields, references, nil
}

func (o *Orm) setFieldsDefaults(f *driver.Fields) error {
	defaults := make(map[int]reflect.Value)
	for ii, v := range f.Tags {
		def := v.Value("default")
		if def == "" {
			continue
		}
		if driver.IsFunc(def) {
			// Currently we only support two hardcoded functions, now() and today()
			fname, _ := driver.SplitFuncArgs(def)
			fn, ok := ormFuncs[fname]
			if !ok {
				return fmt.Errorf("unknown orm function %s()", fname)
			}
			retType := fn.Type().Out(0)
			if retType != f.Types[ii] {
				return fmt.Errorf("type mismatch: orm function %s() returns %s, but field %s in %s is of type %s", fname, retType, f.QNames[ii], f.Type, f.Types[ii])
			}
			if o.driver.Capabilities()&driver.CAP_DEFAULTS == 0 || !o.driver.HasFunc(fname, retType) {
				defaults[ii] = fn
			}
		} else {
			// Raw value, only to be stored in defaults if the driver
			// does not support CAP_DEFAULTS or it's a TEXT value and
			// the driver lacks CAP_DEFAULTS_TEXT
			caps := o.driver.Capabilities()
			if caps&driver.CAP_DEFAULTS != 0 && (caps&driver.CAP_DEFAULTS_TEXT != 0 || !isText(f, ii)) {
				continue
			}
			// Try to parse it
			ftyp := f.Types[ii]
			indirs := 0
			for ftyp.Kind() == reflect.Ptr {
				indirs++
				ftyp = ftyp.Elem()
			}
			val := reflect.New(ftyp)
			if err := input.Parse(def, val.Interface()); err != nil {
				return fmt.Errorf("invalid default value %q for field %s of type %s in %s: %s", def, f.QNames[ii], f.Types[ii], f.Type, err)
			}
			if indirs == 0 {
				defaults[ii] = val.Elem()
			} else {
				// Pointer, need to allocate a new value each time
				typ := f.Types[ii].Elem()
				defVal := val.Elem()
				f := func() reflect.Value {
					v := reflect.New(typ).Elem()
					for v.Kind() == reflect.Ptr {
						v.Set(reflect.New(v.Type().Elem()))
						v = v.Elem()
					}
					v.Set(defVal)
					return v
				}
				defaults[ii] = reflect.ValueOf(f)
			}
		}
	}
	if len(defaults) > 0 {
		f.Defaults = defaults
	}
	return nil
}

func (o *Orm) dtags() []string {
	tags := o.driver.Tags()
	allTags := make([]string, len(tags)+1)
	copy(allTags, tags)
	allTags[len(tags)] = "orm"
	return allTags
}

func (o *Orm) model(obj interface{}) (*model, error) {
	t := reflect.TypeOf(obj)
	if t == nil {
		return nil, errUntypedNilPointer
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	model := o.typeRegistry[t]
	if model == nil {
		return nil, fmt.Errorf("no model registered for type %v with tags %q", t, o.tags)
	}
	return model, nil
}

// NameTable returns the Table for the model with
// the given Name. Note that this is not the table
// name, but model name, provided in the Options.Name field.
// If no Name was provided in the Options for a given
// type, a name is assigned using the following rules:
//
//  - Types in package main use the type name as is.
//	type Something... in package main is named Something
//
//  - Types in non-main packages use the fully qualified type name.
//	type Something... in package foo/bar is named foo/bar.Something
//	type Something... in package example.com/mypkg is named example.com/mypkg.Something
func (o *Orm) NameTable(name string) *Table {
	for _, v := range o.typeRegistry {
		if v.name == name {
			return tableWithModel(v)
		}
	}
	return nil
}

// TypeTable returns the Table for the given type, or
// nil if there's no such table.
func (o *Orm) TypeTable(typ reflect.Type) *Table {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	model := o.typeRegistry[typ]
	if model != nil {
		return tableWithModel(model)
	}
	return nil
}

// returns wheter the kind defaults to nullempty option
func defaultsToNullEmpty(typ reflect.Type, t *structs.Tag) bool {
	if t.Has("references") || t.Has("codec") || t.Has("notnull") {
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

func defaultsToOmitEmpty(typ reflect.Type, t *structs.Tag) bool {
	return t.Has("auto_increment") || t.Has("default")
}

// Returns the default table name for a type
func defaultTableName(typ reflect.Type) string {
	n := typ.Name()
	if p := typ.PkgPath(); !strings.HasPrefix(p, "main") {
		n = strings.Replace(p, "/", "_", -1) + n
	}
	return stringutil.CamelCaseToLower(n, "_")
}

func typeName(typ reflect.Type) string {
	if p := typ.PkgPath(); !strings.HasPrefix(p, "main") {
		return p + "." + typ.Name()
	}
	return typ.Name()
}

func isText(f *driver.Fields, idx int) bool {
	if f.Types[idx].Kind() == reflect.String {
		maxLen, _ := f.Tags[idx].MaxLength()
		fixLen, _ := f.Tags[idx].Length()
		return maxLen == 0 && fixLen == 0
	}
	return false
}

func init() {
	globalRegistry.names = make(map[string]nameRegistry)
	globalRegistry.types = make(map[string]typeRegistry)
}
