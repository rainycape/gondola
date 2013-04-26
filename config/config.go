package config

// Config contains the common fields used to configure a Gondola
// application.
type Config struct {
	Debug         bool   `help:"Enable debug mode"`                // Sets gondola/defaults/SetDebug()
	Port          int    `default:"8888" help:"Port to listen on"` // Sets gondola/defaults/SetPort()
	CacheUrl      string `help:"Default cache URL"`
	Secret        string `help:"Secret used for, among other things, hashing cookies"` // Sets gondola/defaults/SetSecret()
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`     // Sets gondola/defaults/SetMailServer()
	MailServer    string `default:"localhost:25" help:"Default mail server used by gondola/mail"`
}
