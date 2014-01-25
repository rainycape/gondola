package config

const (
	// SET signal is emitted after the configuration has been parsed.
	// The object is the configuration itself.
	SET = "gnd.la/config.set"
)

// Config contains the common fields used to configure a Gondola
// application. Settings alter the values returned by Default*
// functions in gnd.la/app and gnd.la/mail.
// See the help and comment on each field for further information.
type Config struct {
	// AppDebug sets the value returned by gnd.la/app.DefaultAppDebug().
	AppDebug bool `help:"Enable app debug mode. This causes runtime errors to generate a detailed error page"`
	// TemplateDebug sets the value returned by gnd.la/app.DefaultTemplateDebug().
	TemplateDebug bool `help:"Enable template debug mode. This disables asset bundling and template caching"`
	// LogDebug sets the log level on gnd.la/log.Std to DEBUG, rather
	// the INFO.
	LogDebug bool `help:"Set the logging level to debug without enabling debug mode"`
	// Language sets the value returned by gnd.la/app.DefaultLanguage().
	Language string `help:"Set the default language for translating strings"`
	// Port sets the value returned by gnd.la/app.DefaultPort().
	Port int `default:"8888" help:"Port to listen on"`
	// Database sets the value returned by gnd.la/app.DefaultDatabase().
	Database *URL `help:"Default database to use, used by Context.Orm()"`
	// Cache sets the value returned by gnd.la/app.DefaultCache().
	Cache *URL `help:"Default cache, returned by Context.Cache()"`
	// Blobstore sets the value returned by gnd.la/app.DefaultBlobstore().
	Blobstore *URL `help:"Default blobstore, returned by Context.Blobstore()"`
	// Secret sets the value returned by gnd.la/app.DefaultSecret().
	Secret string `help:"Secret used for, among other things, hashing cookies"`
	// EncryptionKey sets the value returned by gnd.la/app.DefaultEncryptionKey().
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`
	// Mailserver sets the value returned by gnd.la/mail.DefaultServer().
	MailServer string `default:"localhost:25" help:"Default mail server used by gnd.la/mail"`
	// FromEmail sets the value returned by gnd.la/mail.DefaultFrom().
	FromEmail string `help:"Default From address when sending emails"`
	// AdminEmail sets the value returned by gnd.la/mail.Admin().
	AdminEmail string `help:"When running in non-debug mode, any error messages will be emailed to this adddress"`
}
