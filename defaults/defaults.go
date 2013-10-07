// Package defaults acts like a global storage for
// default values for other Gondola packages.
// These values might be altered by gnd.la/config
// after successfully calling config.Parse().
// See the documentation on gnd.la/config to
// learn which values alter these defaults.
package defaults

import (
	"gnd.la/config"
	"gnd.la/log"
	"gnd.la/mail"
	"gnd.la/signal"
)

var (
	port                = 8888
	debug               = false
	secret              = ""
	encryptionKey       = ""
	adminEmail          = ""
	errorLoggingEnabled = false
	_database           *config.URL
	_cache              *config.URL
	_blobstore          *config.URL
)

// Port returns the default port used by Mux.
// The default is 8888.
func Port() int {
	return port
}

// SetPort changes the default port.
func SetPort(p int) {
	if p <= 0 {
		log.Panicf("Invalid port number %d. Must be greater than 0", port)
	}
	port = p
}

// Debug returns the default debug value used by
// gnd.la/mux/Mux instances.
func Debug() bool {
	return debug
}

// SetDebug changes the global default for the debug
// value. Setting it to true also changes the gnd.la/log
// level to LDebug (but setting it to false does not alter
// the log level). The default is false.
// Note the this value is set
// during Mux creation, so if you change this value
// it will only affect muxes created after the change.
func SetDebug(d bool) {
	debug = d
	if d {
		log.SetLevel(log.LDebug)
	}
}

// Secret returns the default secret used by
// gnd.la/mux/Mux instances.
func Secret() string {
	return secret
}

// SetSecret changes the global default value for
// the Mux secret.
// Note the this value is set
// during Mux creation, so if you change this value
// it will only affect muxes created after the change.
func SetSecret(s string) {
	secret = s
}

// EncryptionKey returns the default encryption key used
// by gnd.la/mux/Mux instances.
func EncryptionKey() string {
	return encryptionKey
}

// SetEncryptionKey changes the global default value for
// the Mux encryption key.
// Note the this value is set
// during Mux creation, so if you change this value
// it will only affect muxes created after the change.
func SetEncryptionKey(k string) {
	encryptionKey = k
}

// MailServer returns the default mail server URL. Note
// that this function returns gnd.la/mail/DefaultServer(),
// so both functions return the same.
func MailServer() string {
	return mail.DefaultServer()
}

// SetMailServer sets the default mail server URL.
// See the documentation on gnd.la/mail/SetDefaultServer()
// for further details.
func SetMailServer(s string) {
	mail.SetDefaultServer(s)
	enableMailErrorLogging()
}

// FromEmail returns the default From address used
// in outgoing emails. Note that this function returns
// gnd.la/mail/DefaultFrom(), so both functions
// return the same.
func FromEmail() string {
	return mail.DefaultFrom()
}

// SetFromEmail sets the default From address used in
// outgoing emails.
func SetFromEmail(f string) {
	mail.SetDefaultFrom(f)
	enableMailErrorLogging()
}

// AdminEmail returns the administrator's email.
func AdminEmail() string {
	return adminEmail
}

// SetAdminEmail sets the administrator email. If this value
// is set to a non-empty address, DefaultFrom() is non-empty
// and Debug() is false, email error reporting will be enabled
// by sending any logged error message (including unhandled panics) to
// the provided address.
func SetAdminEmail(email string) {
	adminEmail = email
	enableMailErrorLogging()
}

// Database returns the default database configuration URL.
func Database() *config.URL {
	return _database
}

// SetDatabase sets the default database.
func SetDatabase(database *config.URL) {
	_database = database
}

// Cache returns the default cache configuration URL.
func Cache() *config.URL {
	return _cache
}

// SetCache sets the default cache configuration URL.
func SetCache(cache *config.URL) {
	_cache = cache
}

// Blobstore returns the default blobstore configuration URL.
func Blobstore() *config.URL {
	return _blobstore
}

// SetBlobstore sets the default blobstore configuration URL.
func SetBlobstore(blobstore *config.URL) {
	_blobstore = blobstore
}

func enableMailErrorLogging() {
	if !errorLoggingEnabled && !Debug() && MailServer() != "" && FromEmail() != "" && AdminEmail() != "" {
		errorLoggingEnabled = true
		log.Infof("Enabling email error logging to %q via %q", AdminEmail(), MailServer())
		writer := log.NewSmtpWriter(log.LError, MailServer(), FromEmail(), AdminEmail())
		log.Std.AddWriter(writer)
	}
}

func init() {
	signal.MustRegister(signal.CONFIGURED, func(_ string, object interface{}) {
		setDefaults(object)
	})
}
