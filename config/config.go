package config

// Config contains the common fields used to configure a Gondola
// application.
type Config struct {
	Debug         bool   `help:"Enable debug mode"`
	Port          int    `default:"8888" help:"Port to listen on"`
	CacheUrl      string `help:"Default cache URL"`
	Secret        string `help:"Secret used for, among other things, hashing cookies"`
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`
}
