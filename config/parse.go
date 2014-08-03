package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gnd.la/form/input"
	"gnd.la/internal"
	"gnd.la/util/pathutil"
	"gnd.la/util/stringutil"
	"gnd.la/util/types"
)

var (
	DefaultFilename = pathutil.Relative("app.conf")
	configName      *string
)

type fieldValue struct {
	Value reflect.Value
	Tag   reflect.StructTag
}

type fieldMap map[string]*fieldValue

func (f fieldMap) Append(name string, value reflect.Value, tag reflect.StructTag) error {
	if _, ok := f[name]; ok {
		return fmt.Errorf("duplicate field name %q", name)
	}
	f[name] = &fieldValue{value, tag}
	return nil
}

type varMap map[string]interface{}

// Filename returns the filename used by Parse(). If the -config command
// line flag was provided, it returns its value. Otherwise, it returns
// DefaultFilename().
func Filename() string {
	if configName == nil || *configName == "" {
		return DefaultFilename
	}
	return *configName
}

func parameterName(name string) string {
	return stringutil.CamelCaseToLower(name, "-")
}

func hasProvidedConfig() bool {
	var ret bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			ret = true
		}
	})
	return ret
}

func parseValue(v reflect.Value, raw string) error {
	switch v.Type().Kind() {
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(int64(value))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(uint64(value))
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		v.SetFloat(value)
	case reflect.String:
		v.SetString(raw)
	case reflect.Slice:
		fields, err := stringutil.SplitFields(raw, ",")
		if err != nil {
			return fmt.Errorf("error splitting values: %s", err)
		}
		count := len(fields)
		v.Set(reflect.MakeSlice(v.Type(), count, count))
		for ii, f := range fields {
			if err := input.Parse(f, v.Index(ii).Addr().Interface()); err != nil {
				return fmt.Errorf("error parsing field at index %d: %s", ii, err)
			}
		}
	case reflect.Map:
		fields, err := stringutil.SplitFields(raw, ",")
		if err != nil {
			return fmt.Errorf("error splitting values: %s", err)
		}
		v.Set(reflect.MakeMap(v.Type()))
		ktyp := v.Type().Key()
		etyp := v.Type().Elem()
		for ii, field := range fields {
			subFields, err := stringutil.SplitFields(field, "=")
			if err != nil {
				return fmt.Errorf("error splitting key-value %q: %s", field, err)
			}
			k := reflect.New(ktyp)
			if err := input.Parse(subFields[0], k.Interface()); err != nil {
				return fmt.Errorf("error parsing key at index %d: %s", ii, err)
			}
			elem := reflect.New(etyp)
			if len(subFields) < 2 {
				return fmt.Errorf("invalid map field %q", field)
			}
			if err := input.Parse(subFields[1], elem.Interface()); err != nil {
				return fmt.Errorf("error parsing value at index %d: %s", ii, err)
			}
			v.SetMapIndex(k.Elem(), elem.Elem())
		}
	default:
		parser, ok := v.Interface().(input.Parser)
		if !ok {
			return fmt.Errorf("can't parse value of type %s", v.Type())
		}
		if v.Kind() == reflect.Ptr && !v.Elem().IsValid() {
			v.Set(reflect.New(v.Type().Elem()))
			parser = v.Interface().(input.Parser)
		}
		return parser.Parse(raw)
	}
	return nil
}

// ParseFile parses the given config file into the given config
// struct. No signal is emitted. Look at the documentation of
// Parse() for information on the supported types as well as
// the name mangling performed in the struct fields to convert
// them to config file keys.
func ParseFile(filename string, config interface{}) error {
	fields, err := configFields(config)
	if err != nil {
		return err
	}
	return parseFile(filename, fields)
}

// ParseReader parses the config from the given io.Reader into
// the given config struct. No signal is emitted. Look at the documentation
// of Parse() for information on the supported types as well as
// the name mangling performed in the struct fields to convert
// them to config file keys.
func ParseReader(r io.Reader, config interface{}) error {
	fields, err := configFields(config)
	if err != nil {
		return err
	}
	return parseReader(r, fields)
}

func parseFile(filename string, fields fieldMap) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return parseReader(f, fields)
}

func parseReader(r io.Reader, fields fieldMap) error {
	values, err := stringutil.ParseIni(r)
	if err != nil {
		return err
	}
	/* Now iterate over the fields and copy from the map */
	for k, v := range fields {
		name := parameterName(k)
		if raw, ok := values[name]; ok && raw != "" {
			err := parseValue(v.Value, raw)
			if err != nil {
				return fmt.Errorf("error parsing config file field %q (struct field %q): %s", name, k, err)
			}
		}
	}
	return nil
}

func setupFlags(fields fieldMap) (varMap, error) {
	m := make(varMap)
	for k, v := range fields {
		name := parameterName(k)
		help := v.Tag.Get("help")
		var p interface{}
		val := v.Value
		switch val.Type().Kind() {
		case reflect.Bool:
			p = flag.Bool(name, val.Bool(), help)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			p = flag.Int(name, int(val.Int()), help)
		case reflect.Int64:
			p = flag.Int64(name, val.Int(), help)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			p = flag.Uint(name, uint(val.Uint()), help)
		case reflect.Uint64:
			p = flag.Uint64(name, val.Uint(), help)
		case reflect.Float32, reflect.Float64:
			p = flag.Float64(name, val.Float(), help)
		case reflect.String:
			p = flag.String(name, val.String(), help)
		case reflect.Slice, reflect.Array:
			if !canParse(val.Type().Elem()) {
				return nil, fmt.Errorf("config field %q has unsupported type %s", k, val.Type())
			}
			p = flag.String(name, sliceToString(val), help)
		case reflect.Map:
			if !canParse(val.Type().Elem()) || !canParse(val.Type().Key()) {
				return nil, fmt.Errorf("config field %q has unsupported type %s", k, val.Type())
			}
			p = flag.String(name, mapToString(val), help)
		default:
			if _, ok := val.Interface().(input.Parser); !ok {
				return nil, fmt.Errorf("config field %q has unsupported type %s", k, val.Type())
			}
			// Type implements input.Parser, define it as a string
			p = flag.String(name, types.ToString(val.Interface()), help)
		}
		m[name] = p
	}
	return m, nil
}

func copyFlagValues(fields fieldMap, values varMap) error {
	/* Copy only flags which have been set */
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})
	for k, v := range fields {
		name := parameterName(k)
		if !setFlags[name] {
			continue
		}
		val := v.Value
		switch val.Type().Kind() {
		case reflect.Bool:
			value := *(values[name].(*bool))
			val.SetBool(value)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			value := *(values[name].(*int))
			val.SetInt(int64(value))
		case reflect.Int64:
			value := *(values[name].(*int64))
			val.SetInt(value)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			value := *(values[name].(*uint))
			val.SetUint(uint64(value))
		case reflect.Uint64:
			value := *(values[name].(*uint64))
			val.SetUint(value)
		case reflect.Float32, reflect.Float64:
			value := *(values[name].(*float64))
			val.SetFloat(value)
		case reflect.String:
			value := *(values[name].(*string))
			val.SetString(value)
		default:
			if parser, ok := val.Interface().(input.Parser); ok {
				if val.Kind() == reflect.Ptr && !val.Elem().IsValid() {
					val.Set(reflect.New(val.Type().Elem()))
					parser = val.Interface().(input.Parser)
				}
				value := *(values[name].(*string))
				if err := parser.Parse(value); err != nil {
					return err
				}
				break
			}
			return fmt.Errorf("invalid type in config %s (field %q)", val.Type(), k)
		}
	}
	return nil
}

func reflectValue(config interface{}) (reflect.Value, error) {
	value := reflect.ValueOf(config)
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	if !value.CanAddr() {
		return reflect.Value{}, fmt.Errorf("config must be a pointer to a struct (it's %T)", config)
	}
	return value, nil
}

func configFields(config interface{}) (fieldMap, error) {
	val, err := reflectValue(config)
	if err != nil {
		return nil, err
	}
	return configValueFields(val)
}

func configValueFields(value reflect.Value) (fieldMap, error) {
	fields := make(fieldMap)
	valueType := value.Type()
	for ii := 0; ii < value.NumField(); ii++ {
		field := value.Field(ii)
		if field.Type().Kind() == reflect.Struct {
			subfields, err := configFields(field.Addr().Interface())
			if err != nil {
				return nil, err
			}
			for k, v := range subfields {
				err := fields.Append(k, v.Value, v.Tag)
				if err != nil {
					return nil, err
				}
			}
		} else {
			sfield := valueType.Field(ii)
			if def := sfield.Tag.Get("default"); def != "" {
				err := parseValue(field, def)
				if err != nil {
					return nil, fmt.Errorf("error parsing default value for field %q: %s", sfield.Name, err)
				}
			}
			err := fields.Append(sfield.Name, field, sfield.Tag)
			if err != nil {
				return nil, err
			}
		}
	}
	return fields, nil
}

func canParse(typ reflect.Type) bool {
	if types.IsInt(typ) || types.IsUint(typ) || types.IsFloat(typ) {
		return true
	}
	k := typ.Kind()
	if k == reflect.Bool || k == reflect.String {
		return true
	}
	val := reflect.New(typ)
	if _, ok := val.Interface().(input.Parser); ok {
		return true
	}
	return false
}

func quote(v reflect.Value) string {
	s := types.ToString(v.Interface())
	if v.Kind() == reflect.String {
		return fmt.Sprintf("'%s'", s)
	}
	return s
}

func sliceToString(v reflect.Value) string {
	count := v.Len()
	s := make([]string, count)
	for ii := 0; ii < count; ii++ {
		s[ii] = quote(v.Index(ii))
	}
	return strings.Join(s, ", ")
}

func mapToString(v reflect.Value) string {
	count := v.Len()
	s := make([]string, count)
	for ii, key := range v.MapKeys() {
		value := v.MapIndex(key)
		s[ii] = fmt.Sprintf("%s: %s", quote(key), quote(value))
	}
	return strings.Join(s, ", ")
}

// Parse parses all configurations previously registered using Register or RegisterFunc.
// See those functions for information about adding your own configuration parameters.
func Parse() error {
	configName = flag.String("config", DefaultFilename, "Config file name")
	fields := make(fieldMap)
	for _, v := range registry {
		valueFields, err := configValueFields(v.value)
		if err != nil {
			return err
		}
		for k, v := range valueFields {
			fields[k] = v
		}
	}
	/* Setup flags before calling flag.Parse() */
	flagValues, err := setupFlags(fields)
	if err != nil {
		return err
	}
	/* Now parse the flags */
	flag.Parse()
	/* Read config file first */
	if fn := Filename(); fn != "" {
		if err := parseFile(fn, fields); err != nil {
			if hasProvidedConfig() || !os.IsNotExist(err) {
				return err
			}
		}
	}
	/* Command line overrides config file */
	if err := copyFlagValues(fields, flagValues); err != nil {
		return err
	}
	// Call registry functions
	for _, v := range registry {
		if v.f != nil {
			v.f()
		}
	}
	return nil
}

// MustParse works like Parse, but panics if there's an error.
func MustParse() {
	if err := Parse(); err != nil {
		panic(fmt.Errorf("error parsing config %s: %s", Filename(), err))
	}
}

func init() {
	if internal.InAppEngineDevServer() {
		DefaultFilename = pathutil.Relative("dev.conf")
	}
}
