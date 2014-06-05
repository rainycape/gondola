package app

import (
	"gnd.la/config"
)

// Type Config is represents the App configuration.
type Config struct {
	// Debug indicates if debug mode is enabled. If true,
	// runtime errors generate detailed an error page with
	// stack traces and request information.
	Debug bool `help:"Enable app debug mode. This causes runtime errors to generate a detailed error page"`
	// TemplateDebug indicates if the app should handle
	// templates in debug mode. When it's enabled, assets
	// are not bundled and templates are recompiled each
	// time they are loaded.
	TemplateDebug bool `help:"Enable template debug mode. This disables asset bundling and template caching"`
	// Language indicates the language used for
	// translating strings when there's no LanguageHandler
	// or when it returns an empty string.
	Language string `help:"Set the default language for translating strings"`
	// Port indicates the port to listen on.
	Port      int         `default:"8888" help:"Port to listen on"`
	Database  *config.URL `help:"Default database to use, used by Context.Orm()"`
	Cache     *config.URL `help:"Default cache, returned by Context.Cache()"`
	Blobstore *config.URL `help:"Default blobstore, returned by Context.Blobstore()"`
	// Secret indicates the secret associated with the app,
	// which is used for signed cookies. It should be a
	// random string with at least 32 characters.
	// You can use gondola random-string to generate one.
	Secret string `help:"Secret used for, among other things, hashing cookies"`
	// EncriptionKey is the encryption key for used by the
	// app for, among other things, encrypted cookies. It should
	// be a random string of 16 or 24 or 32 characters.
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`
}

var (
	defaultConfig = Config{
		Port: 8888,
	}
)

func init() {
	config.Register(&defaultConfig)
}
