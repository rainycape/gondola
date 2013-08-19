package defaults

import (
	"reflect"
)

func isInt(v reflect.Value) bool {
	k := v.Type().Kind()
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func isUint(v reflect.Value) bool {
	k := v.Type().Kind()
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64
}

func isNonEmptyString(v reflect.Value) bool {
	if v.Type().Kind() == reflect.String {
		return v.String() != ""
	}
	return false
}

func setStringDefault(val reflect.Value, name string, f func(string)) {
	value := val.FieldByName(name)
	if value.IsValid() && value.String() != "" {
		f(value.String())
	}
}

func setDefaults(object interface{}) {
	val := reflect.ValueOf(object)
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if debug := val.FieldByName("Debug"); debug.Type().Kind() == reflect.Bool {
		SetDebug(debug.Bool())
	}
	if port := val.FieldByName("Port"); isInt(port) || isUint(port) {
		var p int
		if isInt(port) {
			p = int(port.Int())
		} else {
			p = int(port.Uint())
		}
		SetPort(p)
	}

	stringDefaults := map[string]func(string){
		"Database":      SetDatabase,
		"Cache":         SetCache,
		"Secret":        SetSecret,
		"EncryptionKey": SetEncryptionKey,
		"MailServer":    SetMailServer,
		"FromEmail":     SetFromEmail,
		"AdminEmail":    SetAdminEmail,
	}
	for k, v := range stringDefaults {
		setStringDefault(val, k, v)
	}
}
