package config

// Config contains the common fields used to configure a Gondola
// application. Some of them modify settings in gondola/defaults.
// See the help and comment on each field for further information.
type Config struct {
	Debug         bool   `help:"Enable debug mode"`                                                                   // Sets gondola/defaults/SetDebug()
	Port          int    `default:"8888" help:"Port to listen on"`                                                    // Sets gondola/defaults/SetPort()
	Database      string `help:"Default database to use, returned by Context.DB()"`                                   // Sets gondola/defaults/SetDatabase()
	Cache         string `help:"Default cache, returned by Context.Cache()"`                                          // Sets gondola/defaults/SetCache()
	Secret        string `help:"Secret used for, among other things, hashing cookies"`                                // Sets gondola/defaults/SetSecret()
	EncryptionKey string `help:"Key used for encryption (e.g. encrypted cookies)"`                                    // Sets gondola/defaults/SetEncryptionKey()
	MailServer    string `default:"localhost:25" help:"Default mail server used by gondola/mail"`                     // Sets gondola/defaults/SetMailServer()
	FromEmail     string `help:"Default From address when sending emails"`                                            // Sets gondola/defaults/SetFromEmail()
	AdminEmail    string `help:"When running in non-debug mode, any error messages will be emailed to this adddress"` // Sets gondola/defaults/SetAdminEmail()
}
