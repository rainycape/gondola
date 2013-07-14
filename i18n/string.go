package i18n

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
	return T(string(s), lang)
}
