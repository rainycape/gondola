package config

import (
	"gnd.la/log"
	"gnd.la/mail"
)

var (
	mailLogEnabled = false
)

// the options in gnd.la/mail are set from this
// package to avoid an import cycle, since config
// imports signal, which imports log, which imports
// mail, so mail can't import config
func setMailConfig(config interface{}, debug bool) {
	fields := map[string]func(string){
		"MailServer": mail.SetDefaultServer,
		"FromEmail":  mail.SetDefaultFrom,
		"AdminEmail": mail.SetAdminEmail,
	}
	values := make(map[string]string)
	for k, v := range fields {
		val := StringValue(config, k, "")
		values[k] = val
		if val != "" {
			v(val)
		}
	}
	if !mailLogEnabled && !debug && values["MailServer"] != "" && values["FromEmail"] != "" && values["AdminEmail"] != "" {
		mailLogEnabled = true
		admin := values["AdminEmail"]
		server := values["MailServer"]
		writer := log.NewSmtpWriter(log.LError, server, values["FromEmail"], admin)
		log.Std.AddWriter(writer)
	}
}
