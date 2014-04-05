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
type String string

// String returns the String as a plain string.
func (s String) String() string {
	return string(s)
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
