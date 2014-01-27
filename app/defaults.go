package app

import (
	"gnd.la/config"
	"gnd.la/signal"
)

var (
	defaultDebug         bool
	defaultTemplateDebug bool
	defaultLanguage      string
	defaultPort          int
	defaultDatabase      *config.URL
	defaultCache         *config.URL
	defaultBlobstore     *config.URL
	defaultSecret        string
	defaultEncryptionKey string
)

// DefaultDebug returns the default value for
// App.Debug in App instances.
func DefaultDebug() bool { return defaultDebug }

// DefaultTemplateDebug returns the default value for
// App.TemplateDebug in App instances.
func DefaultTemplateDebug() bool { return defaultTemplateDebug }

// DefaultLanguage returns the default value for
// App.DefaultLanguage in App instances.
func DefaultLanguage() string { return defaultLanguage }

// DefaultPort returns the default value for
// App.Port in App instances.
func DefaultPort() int { return defaultPort }

func DefaultDatabase() *config.URL { return defaultDatabase }

func DefaultCache() *config.URL { return defaultCache }

func DefaultBlobstore() *config.URL { return defaultBlobstore }

// DefaultSecret returns the default value for
// App.Secret in App instances.
func DefaultSecret() string { return defaultSecret }

// DefaultEncryptionKey returns the default value for
// App.EncryptionKey in App instances.
func DefaultEncryptionKey() string { return defaultEncryptionKey }

func init() {
	signal.MustRegister(config.SET, func(_ string, conf interface{}) {
		defaultDebug = config.BoolValue(conf, "AppDebug", false)
		defaultTemplateDebug = config.BoolValue(conf, "TemplateDebug", false)
		defaultLanguage = config.StringValue(conf, "Language", "")
		defaultPort = config.IntValue(conf, "Port", 8888)
		config.PointerValue(conf, "Database", &defaultDatabase)
		config.PointerValue(conf, "Cache", &defaultCache)
		config.PointerValue(conf, "Blobstore", &defaultBlobstore)
		defaultSecret = config.StringValue(conf, "Secret", "")
		defaultEncryptionKey = config.StringValue(conf, "EncryptionKey", "")
	})
}
