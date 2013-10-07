package defaults

import (
	"gnd.la/config"
	"gnd.la/log"
	"reflect"
)

func isBool(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	return v.Kind() == reflect.Bool
}

func isInt(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	k := v.Type().Kind()
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func isUint(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	k := v.Type().Kind()
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64
}

func isNonEmptyString(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.Type().Kind() == reflect.String {
		return v.String() != ""
	}
	return false
}

func setStringDefault(val reflect.Value, name string, f func(string)) {
	value := val.FieldByName(name)
	if isNonEmptyString(value) {
		f(value.String())
	}
}

func setDefaults(object interface{}) {
	val := reflect.ValueOf(object)
	for val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if debug := val.FieldByName("Debug"); isBool(debug) {
		SetDebug(debug.Bool())
	}
	if logDebug := val.FieldByName("LogDebug"); logDebug.IsValid() && logDebug.Kind() == reflect.Bool {
		if logDebug.Bool() {
			log.SetLevel(log.LDebug)
		}
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
		"Secret":        SetSecret,
		"EncryptionKey": SetEncryptionKey,
		"MailServer":    SetMailServer,
		"FromEmail":     SetFromEmail,
		"AdminEmail":    SetAdminEmail,
	}
	for k, v := range stringDefaults {
		setStringDefault(val, k, v)
	}
	urlDefaults := map[string]func(*config.URL){
		"Database":  SetDatabase,
		"Cache":     SetCache,
		"Blobstore": SetBlobstore,
	}
	for k, v := range urlDefaults {
		value := val.FieldByName(k)
		v(value.Interface().(*config.URL))
	}
}
