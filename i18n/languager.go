package i18n

// Languager is the interface implemented
// by any object which can return a language
// identifier. Valid language identifiers have
// either 2 characters e.g. "es", "en" or
// either 5 e.g. "es_ES", "en_US".
type Languager interface {
	// Language returns the current language.
	Language() string
}
