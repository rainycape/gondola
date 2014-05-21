package i18n

import (
	"gnd.la/util/stringutil"
)

// TranslatableString is the interface implemented
// by strings that can be translated.
type TranslatableString interface {
	TranslatedString(lang Languager) string
}

// String is an alias for string, but variables
// or constants declared with the type String will
// be extracted for translation.
//
// String declarations might include a context by using the |
// character, which can be escaped by \.
// e.g.
//
//  var foo = i18n.String("ctx|str")
//
// Declares a translatable string with context "ctx" and a
// value of "str".
//
//  var bar = i18n.String("ctx\\|str")
//
// Declares a translatable string without context and with
// the value "ctx|str".
//
// String declarations can also include a plural form by adding
// another | separated field.
//
// var hasPlural = i18n.String("ctx|singular|plural")
type String string

// String returns the String as a plain string.
func (s String) String() string {
	return string(s)
}

// Context returns the translation context for the String, which
// might be empty.
func (s String) Context() string {
	fields, _ := stringutil.SplitFields(string(s), "|")
	if len(fields) > 1 {
		return fields[0]
	}
	return ""
}

// TranslatedString returns the string translated into
// the language returned by lang.
func (s String) TranslatedString(lang Languager) string {
	if string(s) == "" {
		return ""
	}
	fields, _ := stringutil.SplitFields(string(s), "|")
	if len(fields) > 1 {
		return Tc(lang, fields[0], fields[1])
	}
	if len(fields) > 0 {
		return T(lang, fields[0])
	}
	return T(lang, string(s))
}
