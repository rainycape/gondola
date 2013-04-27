package config

import (
	"gondola/defaults"
	"reflect"
)

func isInt(v *fieldValue) bool {
	k := v.Value.Type().Kind()
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func isUint(v *fieldValue) bool {
	k := v.Value.Type().Kind()
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64
}

func isNonEmptyString(v *fieldValue) bool {
	if v.Value.Type().Kind() == reflect.String {
		return v.Value.String() != ""
	}
	return false
}

func setDefaults(fields fieldMap) {
	if debug, ok := fields["Debug"]; ok && debug.Value.Type().Kind() == reflect.Bool {
		defaults.SetDebug(debug.Value.Bool())
	}
	if port, ok := fields["Port"]; ok && (isInt(port) || isUint(port)) {
		var p int
		if isInt(port) {
			p = int(port.Value.Int())
		} else {
			p = int(port.Value.Uint())
		}
		defaults.SetPort(p)
	}
	if secret, ok := fields["Secret"]; ok && isNonEmptyString(secret) {
		defaults.SetSecret(secret.Value.String())
	}
	if encryptionKey, ok := fields["EncryptionKey"]; ok && isNonEmptyString(encryptionKey) {
		defaults.SetEncryptionKey(encryptionKey.Value.String())
	}
	if mailServer, ok := fields["MailServer"]; ok && isNonEmptyString(mailServer) {
		defaults.SetMailServer(mailServer.Value.String())
	}
	if fromEmail, ok := fields["FromEmail"]; ok && isNonEmptyString(fromEmail) {
		defaults.SetFromEmail(fromEmail.Value.String())
	}
	if adminEmail, ok := fields["AdminEmail"]; ok && isNonEmptyString(adminEmail) {
		defaults.SetAdminEmail(adminEmail.Value.String())
	}
}
