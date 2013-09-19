package config

// Config contains the common fields used to configure a Gondola
// application. Some of them modify settings in gondola/defaults.
// See the help and comment on each field for further information.
type Config struct {
	// Sets gondola/defaults/SetDebug()
	Debug bool `help:"Enable debug mode"`
	// If true, sets the log level to DEBUG, without turning on
	// all the debug machinery (special error pages, etc...). Useful
	// for debugging issues in production.
	LogDebug bool `help:"Set the logging level to debug without enabling debug mode"`
	// Sets gondola/defaults/SetPort()
	Port int `default:"8888" help:"Port to listen on"`
	// Sets gondola/defaults/SetDatabase()
	Database string `help:"Default database to use, used by Context.ORM()"`
	// Sets gondola/defaults/SetCache()
	Cache string `help:"Default cache, returned by Context.Cache()"`
	// Sets gondola/defaults/SetSecret()
	Secret string `help:"Secret used for, among other things, hashing cookies"`
	// Sets gondola/defaults/SetEncryptionKey()
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`
	// Sets gondola/defaults/SetMailServer()
	MailServer string `default:"localhost:25" help:"Default mail server used by gondola/mail"`
	// Sets gondola/defaults/SetFromEmail()
	FromEmail string `help:"Default From address when sending emails"`
	// Sets gondola/defaults/SetAdminEmail()
	AdminEmail string `help:"When running in non-debug mode, any error messages will be emailed to this adddress"`
}
