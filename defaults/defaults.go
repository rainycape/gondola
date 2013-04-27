// Package defaults acts like a global storage for
// default values for other Gondola packages.
// These values might be altered by gondola/config
// after successfully calling config.Parse().
// See the documentation on gondola/config to
// learn which values alter these defaults.
package defaults

import (
	"gondola/log"
	"gondola/mail"
)

var (
	port                = 8888
	debug               = false
	secret              = ""
	encryptionKey       = ""
	adminEmail          = ""
	errorLoggingEnabled = false
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
// gondola/mux/Mux instances.
func Debug() bool {
	return debug
}

// SetDebug changes the global default for the debug
// value. Setting it to true also changes the gondola/log
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
// gondola/mux/Mux instances.
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
// by gondola/mux/Mux instances.
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
// that this function returns gondola/mail/DefaultServer(),
// so both functions return the same.
func MailServer() string {
	return mail.DefaultServer()
}

// SetMailServer sets the default mail server URL.
// See the documentation on gondola/mail/SetDefaultServer()
// for further details.
func SetMailServer(s string) {
	mail.SetDefaultServer(s)
	enableMailErrorLogging()
}

// FromEmail returns the default From address used
// in outgoing emails. Note that this function returns
// gondola/mail/DefaultFrom(), so both functions
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

func enableMailErrorLogging() {
	if !errorLoggingEnabled && !Debug() && MailServer() != "" && FromEmail() != "" && AdminEmail() != "" {
		errorLoggingEnabled = true
		log.Infof("Enabling email error logging to %q via %q", AdminEmail(), MailServer())
		writer := log.NewSmtpWriter(log.LError, MailServer(), FromEmail(), AdminEmail())
		log.Std.AddWriter(writer)
	}
}
