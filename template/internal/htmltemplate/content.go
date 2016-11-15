// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"html/template"
	"reflect"
)

type ContentType uint8

const (
	ContentTypePlain ContentType = iota
	ContentTypeCSS
	ContentTypeHTML
	ContentTypeHTMLAttr
	ContentTypeJS
	ContentTypeJSStr
	ContentTypeURL
	// contentTypeUnsafe is used in attr.go for values that affect how
	// embedded content and network messages are formed, vetted,
	// or interpreted; or which credentials network messages carry.
	contentTypeUnsafe
)

type contentTyper interface {
	String() string
	ContentType() ContentType
}

// indirect returns the value, after dereferencing as many times
// as necessary to reach the base type (or nil).
func indirect(a interface{}) interface{} {
	if a == nil {
		return nil
	}
	if t := reflect.TypeOf(a); t.Kind() != reflect.Ptr {
		// Avoid creating a reflect.Value if it's not a pointer.
		return a
	}
	v := reflect.ValueOf(a)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

var (
	errorType       = reflect.TypeOf((*error)(nil)).Elem()
	fmtStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

// indirectToStringerOrError returns the value, after dereferencing as many times
// as necessary to reach the base type (or nil) or an implementation of fmt.Stringer
// or error,
func indirectToStringerOrError(a interface{}) interface{} {
	if a == nil {
		return nil
	}
	v := reflect.ValueOf(a)
	for !v.Type().Implements(fmtStringerType) && !v.Type().Implements(errorType) && v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

// stringify converts its arguments to a string and the type of the content.
// All pointers are dereferenced, as in the text/template package.
func stringify(args ...interface{}) (string, ContentType) {
	if len(args) == 1 {
		switch s := indirect(args[0]).(type) {
		case string:
			return s, ContentTypePlain
		case template.CSS:
			return string(s), ContentTypeCSS
		case template.HTML:
			return string(s), ContentTypeHTML
		case template.HTMLAttr:
			return string(s), ContentTypeHTMLAttr
		case template.JS:
			return string(s), ContentTypeJS
		case template.JSStr:
			return string(s), ContentTypeJSStr
		case template.URL:
			return string(s), ContentTypeURL
		case contentTyper:
			return s.String(), s.ContentType()
		}
	}
	for i, arg := range args {
		args[i] = indirectToStringerOrError(arg)
	}
	return fmt.Sprint(args...), ContentTypePlain
}
