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

func setStringDefault(fields fieldMap, name string, f func(string)) {
	if value, ok := fields[name]; ok && isNonEmptyString(value) {
		f(value.Value.String())
	}
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

	stringDefaults := map[string]func(string){
		"Database":      defaults.SetDatabase,
		"Cache":         defaults.SetCache,
		"Secret":        defaults.SetSecret,
		"EncryptionKey": defaults.SetEncryptionKey,
		"MailServer":    defaults.SetMailServer,
		"FromEmail":     defaults.SetFromEmail,
		"AdminEmail":    defaults.SetAdminEmail,
	}
	for k, v := range stringDefaults {
		setStringDefault(fields, k, v)
	}
}
