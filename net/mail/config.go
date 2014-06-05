package mail

import (
	"gnd.la/config"
)

// Config specifies the mail configuration. It's not recommended
// to change this fields manually. Instead, use their respective
// config keys or flags. See DefaultServer, DefaultFrom and AdminEmail.
var Config struct {
	MailServer  string `default:"localhost:25" help:"Default mail server used by gnd.la/net/mail"`
	DefaultFrom string `help:"Default From address when sending emails"`
	AdminEmail  string `help:"When running in non-debug mode, any error messages will be emailed to this adddress"`
}

func init() {
	Config.MailServer = "localhost:25"
	config.Register(&Config)
}
