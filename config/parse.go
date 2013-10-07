package config

import (
	"flag"
	"fmt"
	"gnd.la/log"
	"gnd.la/signal"
	"gnd.la/types"
	"gnd.la/util"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	defaultFilename = util.RelativePath("conf/current.conf")
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
// The initial value is the path conf/current.conf relative to the application
// binary.
func DefaultFilename() string {
	return defaultFilename
}

// Filename returns the current filename used by Parse(). If the -config command
// line flag was provided, it returns its value. Otherwise, it returns
// DefaultFilename().
func Filename() string {
	if configName == nil {
		return defaultFilename
	}
	return *configName
}

func fileParameterName(name string) string {
	return util.CamelCaseToLower(name, "_")
}

func flagParameterName(name string) string {
	return util.CamelCaseToLower(name, "-")
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
	default:
		parser, ok := v.Interface().(types.Parser)
		if !ok {
			return fmt.Errorf("can't parse value of type %s", v.Type())
		}
		if v.Kind() == reflect.Ptr && !v.Elem().IsValid() {
			v.Set(reflect.New(v.Type().Elem()))
			parser = v.Interface().(types.Parser)
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

// ParseReader parses the config rom the given io.Reader into
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
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	/* Copy strings to a map */
	values := make(map[string]string)
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				values[key] = value
			}
		}
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
		default:
			if _, ok := val.Interface().(types.Parser); !ok {
				return nil, fmt.Errorf("config field %q has unsupported type %s", k, val.Type())
			}
			// Type implements types.Parser, define it as a string
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

// Parse parses the application configuration into the given config struct. If
// the configuration is parsed successfully, the signal signal.CONFIGURED is
// emitted with the given config as its object (which sets the parameters
// in gnd.la/defaults). Check the documentation on the gnd.la/signal package
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
	signal.Emit(signal.CONFIGURED, config)
	return nil
}

// MustParse works like Parse, but panics if there's an error.
func MustParse(config interface{}) {
	err := Parse(config)
	if err != nil {
		log.Panicf("error parsing config: %s", err)
	}
}
