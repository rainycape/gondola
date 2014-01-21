// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"html/template"
	"reflect"
	"strconv"
)

type contentType uint8

const (
	contentTypePlain contentType = iota
	contentTypeCSS
	contentTypeHTML
	contentTypeHTMLAttr
	contentTypeJS
	contentTypeJSStr
	contentTypeURL
	// contentTypeUnsafe is used in attr.go for values that affect how
	// embedded content and network messages are formed, vetted,
	// or interpreted; or which credentials network messages carry.
	contentTypeUnsafe
)

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
func stringify(args ...interface{}) (string, contentType) {
	if len(args) == 1 {
		v := indirect(args[0])
		switch s := v.(type) {
		case string:
			return s, contentTypePlain
		case template.CSS:
			return string(s), contentTypeCSS
		case template.HTML:
			return string(s), contentTypeHTML
		case template.HTMLAttr:
			return string(s), contentTypeHTMLAttr
		case template.JS:
			return string(s), contentTypeJS
		case template.JSStr:
			return string(s), contentTypeJSStr
		case template.URL:
			return string(s), contentTypeURL
		case int:
			// return contentTypeHTML, since this does not
			// need escaping and is the most common context
			// in templates
			return strconv.Itoa(s), contentTypeHTML
		case float64:
			return strconv.FormatFloat(s, 'g', -1, 64), contentTypeHTML
		}
		return fmt.Sprint(indirectToStringerOrError(v)), contentTypePlain
	}
	for i, arg := range args {
		val := indirectToStringerOrError(arg)
		if val != nil {
			args[i] = val
		} else {
			args[i] = ""
		}
	}
	return fmt.Sprint(args...), contentTypePlain
}
