// Package mail provides a conveniency interface over net/smtp, to
// facilitate the most common tasks when sending emails.
package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"gnd.la/util"
	"io"
	"io/ioutil"
	"net/smtp"
	"strings"
	"text/template"
)

var (
	defaultServer = "localhost:25"
	defaultFrom   = ""
)

type Headers map[string]string

type Attachment struct {
	name        string
	contentType string
	data        []byte
}

// NewAttachment returns a new attachment which can be passed
// to Send() and SendVia(). If contentType is empty, it defaults
// to application/octet-stream
func NewAttachment(name, contentType string, r io.Reader) (*Attachment, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = "file"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &Attachment{name, contentType, data}, nil
}

// SendVia sends an email using the specified server from the specified address
// to the given addresses (separated by commmas). Attachments and addittional
// headers might be specified, like Subject or Reply-To. To include authentication info,
// embed it into the server address (e.g. user@gmail.com:patata@smtp.gmail.com).
// If you want to use CRAM authentication, prefix the username with cram?
// (e.g. cram?pepe:12345@example.com), otherwise PLAIN is used.
// If server is empty, it defaults to localhost:25. If from is empty, DefaultFrom()
// is used in its place.
func SendVia(server, from, to, message string, headers Headers, attachments []*Attachment) error {
	if server == "" {
		server = "localhost:25"
	}
	if from == "" {
		from = defaultFrom
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
	buf := bytes.NewBuffer(nil)
	if headers == nil {
		headers = make(Headers)
		headers["To"] = to
	}
	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	if len(attachments) > 0 {
		boundary := "Gondola-Boundary-" + util.RandomString(16)
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		buf.WriteString("--" + boundary + "\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString(message)
		for _, v := range attachments {
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
		buf.WriteString(message)
	}
	return smtp.SendMail(server, auth, from, strings.Split(to, ","), buf.Bytes())
}

// Send works like SendVia(), but uses the mail server
// specified by DefaultServer()
func Send(from, to, message string, headers Headers, attachments []*Attachment) error {
	return SendVia(defaultServer, from, to, message, headers, attachments)
}

// SendTemplateVia parses the template from templateFile, executes it
// with the data argument and then sends the resulting string as
// the message using SendVia.
func SendTemplateVia(server, from, to, templateFile string, data interface{}, headers Headers, attachments []*Attachment) error {
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}
	return SendVia(server, from, to, buf.String(), headers, attachments)
}

// SendTemplate works like SendTemplateVia, but uses the mail
// server specified by DefaultServer()
func SendTemplate(from, to, templateFile string, data interface{}, headers Headers, attachments []*Attachment) error {
	return SendTemplateVia(defaultServer, from, to, templateFile, data, headers, attachments)
}

// DefaultServer returns the default mail server URL.
func DefaultServer() string {
	return defaultServer
}

// SetDefaultServer sets the default mail server URL.
// See the documentation on SendVia()
// for further information (authentication, etc...).
// The default value is localhost:25.
func SetDefaultServer(s string) {
	if s == "" {
		s = "localhost:25"
	}
	defaultServer = s
}

// DefaultFrom returns the default From address used
// in outgoing emails.
func DefaultFrom() string {
	return defaultFrom
}

// SetDefaultFrom sets the default From address used
// in outgoing emails.
func SetDefaultFrom(f string) {
	defaultFrom = f
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
