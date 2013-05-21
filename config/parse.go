package config

import (
	"flag"
	"fmt"
	"gondola/log"
	"gondola/util"
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
		return fmt.Errorf("Duplicate field name %q", name)
	}
	f[name] = &fieldValue{value, tag}
	return nil
}

type varMap map[string]interface{}

func SetDefaultFilename(name string) {
	defaultFilename = name
}

func DefaultFilename() string {
	return defaultFilename
}

func Filename() string {
	if configName == nil {
		return defaultFilename
	}
	return *configName
}

func fileParameterName(name string) string {
	return util.UnCamelCase(name, "_")
}

func flagParameterName(name string) string {
	return util.UnCamelCase(name, "-")
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
		return fmt.Errorf("Can't parse values of type %q", v.Type().Name())
	}
	return nil
}

func parseConfigFile(r io.Reader, fields fieldMap) error {
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
				return fmt.Errorf("Error parsing config file field %q (struct field %q): %s", name, k, err)
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
			return nil, fmt.Errorf("Invalid type in config %s (field %s)", val.Type().Name(), k)
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
			return fmt.Errorf("Invalid type in config %q (field %q)", val.Type().Name(), k)
		}
	}
	return nil
}

func configFields(value reflect.Value) (fieldMap, error) {
	fields := make(fieldMap)
	valueType := value.Type()
	for ii := 0; ii < value.NumField(); ii++ {
		field := value.Field(ii)
		if field.Type().Kind() == reflect.Struct {
			subfields, err := configFields(field)
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
					return nil, fmt.Errorf("Error parsing default value for field %q: %s", sfield.Name, err)
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

func Parse(config interface{}) error {
	configName = flag.String("config", defaultFilename, "Config file name")
	value := reflect.ValueOf(config)
	for kind := value.Kind(); kind == reflect.Interface || kind == reflect.Ptr; {
		value = value.Elem()
		kind = value.Kind()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("Config must be a struct or pointer to struct (it's %T)", config)
	}
	fields, err := configFields(value)
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
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer f.Close()
		err = parseConfigFile(f, fields)
		if err != nil {
			return err
		}
	}
	/* Command line overrides config file */
	err = copyFlagValues(fields, flagValues)
	if err != nil {
		return err
	}
	setDefaults(fields)
	return nil
}

func MustParse(config interface{}) {
	err := Parse(config)
	if err != nil {
		log.Panicf("Error parsing config: %s", err)
	}
}
