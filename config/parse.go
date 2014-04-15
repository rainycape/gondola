package config

import (
	"flag"
	"fmt"
	"gnd.la/form/input"
	"gnd.la/internal"
	"gnd.la/log"
	"gnd.la/signal"
	"gnd.la/util/pathutil"
	"gnd.la/util/stringutil"
	"gnd.la/util/types"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const DefaultName = "app.conf"

var (
	defaultFilename = pathutil.Relative(DefaultName)
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

// SetDefaultFilename changes the default filename used by Parse().
func SetDefaultFilename(name string) {
	defaultFilename = name
}

// DefaultFilename returns the default config filename used by Parse().
// It might changed by calling SetDefaultFilename() or overriden using
// the -config command line flag (the latter, of present, takes precendence).
// The initial value is app.conf in the same directory as the application
// binary.
func DefaultFilename() string {
	return defaultFilename
}

// Filename returns the filename used by Parse(). If the -config command
// line flag was provided, it returns its value. Otherwise, it returns
// DefaultFilename().
func Filename() string {
	if configName == nil || *configName == "" {
		return defaultFilename
	}
	return *configName
}

func fileParameterName(name string) string {
	return stringutil.CamelCaseToLower(name, "_")
}

func flagParameterName(name string) string {
	return stringutil.CamelCaseToLower(name, "-")
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
		name := fileParameterName(k)
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
		name := flagParameterName(k)
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
		name := flagParameterName(k)
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
			return fmt.Errorf("invalid type in config %q (field %q)", val.Type().Name(), k)
		}
	}
	return nil
}

func configFields(config interface{}) (fieldMap, error) {
	value := reflect.ValueOf(config)
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	if !value.CanAddr() {
		return nil, fmt.Errorf("config must be a pointer to a struct (it's %T)", config)
	}
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

// Parse parses the application configuration into the given config struct. If
// the configuration is parsed successfully, the signal SET is
// emitted with the given config as its object (which sets the default parameters
// in gnd.la/app and gnd.la/mail, among other packages.
// Check the documentation on the gnd.la/signal package
// to learn more about Gondola's signals.
//
// Supported types include bool, string, u?int(|8|6|32|62) and float(32|64). If
// any config field type is not supported, an error is returned. Additionally,
// two struct tags are taken into account. The "help" tag is used when to provide
// a help string to the user when defining command like flags, while the "default"
// tag is used to provide a default value for the field in case it hasn't been
// provided as a config key nor a command line flag.
//
// The parsing process starts by reading the config file returned by Filename()
// (which might be overriden by the -config command line flag), and then parses
// any flags provided in the command line. This means any value in the config
// file might be overriden by a command line flag.
//
// Go's idiomatic camel-cased struct field names are mangled into lowercase words
// to produce the flag names and config fields. e.g. a field named "FooBar" will
// produce a "-foo-bar" flag and a "foo_bar" config key. Embedded struct are
// flattened, as if their fields were part of the container struct. Finally, while
// not mandatory, is very recommended that your config struct embeds config.Config,
// so the standard parameters for Gondola applications are already defined for you.
// e.g.
//
//  var MyConfig struct {
//	config.Config
//	MyStringValue	string
//	MyINTValue	int `help:"Some int used for something" default:"42"`
//  }
//
//  func init() {
//	config.MustParse(&MyConfig)
//  }
//  // Besides the Gondola's standard flags and keys, this config would define
//  // the flags -my-string-value and -my-int-value as well as the config file keys
//  // my_string_value and my_int_value.
//
func Parse(config interface{}) error {
	configName = flag.String("config", defaultFilename, "Config file name")
	fields, err := configFields(config)
	if err != nil {
		return err
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
			return err
		}
	}
	/* Command line overrides config file */
	if err := copyFlagValues(fields, flagValues); err != nil {
		return err
	}
	Set(config)
	return nil
}

// MustParse works like Parse, but panics if there's an error.
func MustParse(config interface{}) {
	err := Parse(config)
	if err != nil {
		panic(fmt.Errorf("error parsing config: %s", err))
	}
}

// Set sets the given value as the main config. It will emit the
// SET signal, so listeners will be aware of the
// configure. You only need to call this function if, for some
// reason, you're not using Parse().
func Set(config interface{}) {
	debug := BoolValue(config, "LogDebug", false)
	if debug {
		log.SetLevel(log.LDebug)
	}
	debug = debug && BoolValue(config, "AppDebug", false)
	setMailConfig(config, debug)
	signal.Emit(SET, config)
}

func init() {
	if internal.InAppEngineDevServer() {
		defaultFilename = pathutil.Relative("dev.conf")
	}
}
