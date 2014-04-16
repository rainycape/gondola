// +build !appengine

package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"

	"gnd.la/util/stringutil"
)

func sendMail(to []string, msg *Message) error {
	from := defaultFrom
	server := defaultServer
	if msg.Server != "" {
		server = msg.Server
	}
	if msg.From != "" {
		from = msg.From
	}
	if from == "" {
		return errNoFrom
	}
	var auth smtp.Auth
	cram, username, password, server := parseServer(server)
	if username != "" || password != "" {
		if cram {
			auth = smtp.CRAMMD5Auth(username, password)
		} else {
			auth = smtp.PlainAuth("", username, password, server)
		}
	}
	var buf bytes.Buffer
	headers := msg.Headers
	if headers == nil {
		headers = make(Headers)
	}
	if msg.Subject != "" {
		headers["Subject"] = msg.Subject
	}
	headers["From"] = from
	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	body, err := messageBody(msg)
	if err != nil {
		return err
	}
	if len(msg.Attachments) > 0 {
		boundary := "Gondola-Boundary-" + stringutil.Random(16)
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		buf.WriteString("--" + boundary + "\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString(body)
		for _, v := range msg.Attachments {
			buf.WriteString("\r\n\r\n--" + boundary + "\r\n")
			buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", v.contentType))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n\r\n", v.name))

			b := make([]byte, base64.StdEncoding.EncodedLen(len(v.data)))
			base64.StdEncoding.Encode(b, v.data)
			buf.Write(b)
			buf.WriteString("\r\n--" + boundary)
		}
		buf.WriteString("--")
	} else {
		buf.Write([]byte{'\r', '\n'})
		buf.WriteString(body)
	}
	if server == "echo" {
		printer("To: %s\n\n%s\n", strings.Join(to, ", "), buf.String())
		return nil
	}
	return smtp.SendMail(server, auth, from, to, buf.Bytes())
}

func parseServer(server string) (bool, string, string, string) {
	// Check if the server includes authentication info
	cram := false
	var username string
	var password string
	if idx := strings.LastIndex(server, "@"); idx >= 0 {
		var credentials string
		credentials, server = server[:idx], server[idx+1:]
		if strings.HasPrefix(credentials, "cram?") {
			credentials = credentials[5:]
			cram = true
		}
		colon := strings.Index(credentials, ":")
		if colon >= 0 {
			username = credentials[:colon]
			if colon < len(credentials)-1 {
				password = credentials[colon+1:]
			}
		} else {
			username = credentials
		}
	}
	return cram, username, password, server
}
