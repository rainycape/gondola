package config

// Config contains the common fields used to configure a Gondola
// application. Some of them modify settings in gnd.la/defaults.
// See the help and comment on each field for further information.
type Config struct {
	// Sets gnd.la/defaults/SetDebug()
	Debug bool `help:"Enable debug mode"`
	// If true, sets the log level to DEBUG, without turning on
	// all the debug machinery (special error pages, etc...). Useful
	// for debugging issues in production.
	LogDebug bool `help:"Set the logging level to debug without enabling debug mode"`
	// Sets gnd.la/defaults/SetPort()
	Port int `default:"8888" help:"Port to listen on"`
	// Sets gnd.la/defaults/SetDatabase()
	Database *URL `help:"Default database to use, used by Context.ORM()"`
	// Sets gnd.la/defaults/SetCache()
	Cache *URL `help:"Default cache, returned by Context.Cache()"`
	// Sets gnd.la/defaults/SetBlobstore()
	Blobstore *URL `help:"Default blobstore, returned by Context.Store()"`
	// Sets gnd.la/defaults/SetSecret()
	Secret string `help:"Secret used for, among other things, hashing cookies"`
	// Sets gnd.la/defaults/SetEncryptionKey()
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`
	// Sets gnd.la/defaults/SetMailServer()
	MailServer string `default:"localhost:25" help:"Default mail server used by gnd.la/mail"`
	// Sets gnd.la/defaults/SetFromEmail()
	FromEmail string `help:"Default From address when sending emails"`
	// Sets gnd.la/defaults/SetAdminEmail()
	AdminEmail string `help:"When running in non-debug mode, any error messages will be emailed to this adddress"`
}
