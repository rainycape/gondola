package log

import (
	"fmt"
	"os"

	"gnd.la/net/mail"
)

type SmtpWriter struct {
	level  LLevel
	server string
	from   string
	to     []string
}

func (w *SmtpWriter) Level() LLevel {
	return w.level
}

func (w *SmtpWriter) Write(level LLevel, flags int, b []byte) (int, error) {
	if w.server == "" || len(w.to) == 0 {
		return 0, nil
	}

	hostname, _ := os.Hostname()
	subject := fmt.Sprintf("%s message on %s", level.String(), hostname)
	err := mail.Send(&mail.Message{
		To:       w.to,
		Subject:  subject,
		TextBody: string(b),
	})
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func NewSmtpWriter(level LLevel, server, from, to string) *SmtpWriter {
	addrs := mail.MustParseAddressList(to)
	return &SmtpWriter{level, server, from, addrs}
}
