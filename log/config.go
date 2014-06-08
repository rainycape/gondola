package log

import (
	"gnd.la/config"
	"gnd.la/net/mail"
)

var logConfig struct {
	LogDebug bool
}

func init() {
	config.RegisterFunc(&logConfig, func() {
		if logConfig.LogDebug {
			Std.SetLevel(LDebug)
		} else {
			// Check if we should send errors to the admin email
			admin := mail.AdminEmail()
			from := mail.DefaultFrom()
			server := mail.DefaultServer()
			if admin != "" && server != "" {
				writer := NewSmtpWriter(LError, server, from, admin)
				Std.AddWriter(writer)
			}
		}
	})
}
