package config

import (
	"reflect"
)

var (
	registry []*entry
)

type entry struct {
	value reflect.Value
	f     func()
}

// Register is a shorthand for RegisterFunc(value, nil).
func Register(value interface{}) {
	RegisterFunc(value, nil)
}

// Register adds a struct pointer to be parsed by the application configuration.
//
// Supported field types include bool, string, u?int(|8|6|32|62) and float(32|64). If
// any config field type is not supported, Register will panic. Additionally,
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
// flattened, as if their fields were part of the container struct.
//
//  var MyConfig struct {
//	MyStringValue	string
//	MyINTValue	int `help:"Some int used for something" default:"42"`
//  }
//
//  func init() {
//	config.Register(&MyConfig)
//	config.MustParse()
//  }
//  // This config would define the flags -my-string-value and -my-int-value
//  // as well as the config file keys my_string_value and my_int_value.
//
// Note that registering several structs with the same field names will cause
// Parse to return an error. Gondola itself registers a few flags. To see them
// all, start your app with the -h flag (e.g. ./myapp -h on Unix or myapp.exe
// -h on Windows).
func RegisterFunc(value interface{}, f func()) {
	val, err := reflectValue(value)
	if err != nil {
		panic(err)
	}
	registry = append(registry, &entry{
		value: val,
		f:     f,
	})
}
