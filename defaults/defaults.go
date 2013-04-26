// Package defaults acts like a global storage for
// default values for other Gondola packages.
// These values might be altered by gondola/config
// after successfully calling config.Parse().
// See the documentation on gondola/config to
// learn which values alter these defaults.
package defaults

import (
	"gondola/log"
)

var (
	port          = 8888
	debug         = false
	secret        = ""
	encryptionKey = ""
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
