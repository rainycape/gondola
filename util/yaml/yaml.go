// Package yaml provides functions for encoding/decoding YAML.
//
// This package is a small wrapper for other packages or packages
// which do the heavy lifting. The purpose of this package is providing
// a stable and easy to find YAML API in Gondola, so users don't have
// to search and evaluate the miriad of different alternatives available
// when it comes to parsing YAML in Go.
package yaml

import (
	"gopkg.in/yaml.v1"
	"io"
	"io/ioutil"
	"os"
)

// Marshal returns the YAML encoding of the in argument. Unexported field names are ignored,
// while exported files are serialized using their name in lowercase as the key. The "yaml"
// struct tag might be used to control the serialization, using the following format:
//
//  `yaml:"[name][,flag1][,flag2][,flagn]"
//
// Supported flags are:
//
//  - omitempty	    Ignore the field if it's not set the zero for its type,
//		    or if it's an empty slice or map
//
//  - flow	    Marshal using a flow style, using explicit indicatos rather than
//		    indentation to denote scope (this only applies for structs, slices,
//		    arrays and maps).
//
//  - inline	    Inline the field it's applied to, so its fields
//		    are processed as if they were part of the outer struct.
//
// Additionally, if the name of the field is set to "-", the field will be skipped.
func Marshal(in interface{}) ([]byte, error) {
	return yaml.Marshal(in)
}

// Unmarshal parses the YAML-encoded in data and decodes it into the value pointed by out.
// See Marshal for the available struct field tags.
func Unmarshal(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, out)
}

// UnmarshalReader works like Unmarshal, but accepts an io.Reader rather
// than the encoded YAML data.
func UnmarshalReader(r io.Reader, out interface{}) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return Unmarshal(data, out)
}

// UnmarshalFile works like Unmarshal, but accepts a filename rather
// than the encoded YAML data.
func UnmarshalFile(filename string, out interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	err = UnmarshalReader(f, out)
	f.Close()
	return err
}
