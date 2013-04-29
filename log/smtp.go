package log

import (
	"fmt"
	"gondola/mail"
	"os"
)

type SmtpWriter struct {
	level  LLevel
	server string
	from   string
	to     string
}

func (w *SmtpWriter) Level() LLevel {
	return w.level
}

func (w *SmtpWriter) Write(level LLevel, flags int, b []byte) (int, error) {
	if w.server == "" || w.from == "" || w.to == "" {
		return 0, nil
	}

	hostname, _ := os.Hostname()
	subject := fmt.Sprintf("%s message on %s", level.String(), hostname)
	err := mail.SendVia(w.server, w.from, w.to, string(b), mail.Headers{"Subject": subject}, nil)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func NewSmtpWriter(level LLevel, server, from, to string) *SmtpWriter {
	return &SmtpWriter{level, server, from, to}
}
