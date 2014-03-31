// Package mail provides a conveniency interface over net/smtp, to
// facilitate the most common tasks when sending emails.
package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"gnd.la/util"
	"gnd.la/util/generic"
	"io"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"strings"
)

var (
	defaultServer = "localhost:25"
	defaultFrom   = ""
	admin         = ""
	errNoMessage  = errors.New("no message specified")
	errNoFrom     = errors.New("missing From: address")
	// Changed for tests
	printer = fmt.Printf
)

// ParseAddressList splits a comma separated list of addresses
// into multiple email addresses. The returned addresses can be
// used to call Send().
func ParseAddressList(s string) ([]string, error) {
	addrs, err := mail.ParseAddressList(s)
	if err != nil {
		return nil, err
	}
	return generic.Map(addrs, func(addr *mail.Address) string {
		if addr.Name != "" {
			return fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
		}
		return addr.Address
	}).([]string), nil
}

// MustParseAddressList works like ParseAddressList, but panics
// if there's an error.
func MustParseAddressList(s string) []string {
	addrs, err := ParseAddressList(s)
	if err != nil {
		panic(err)
	}
	return addrs
}

// Headers represent additional headers to be added to
// the email like e.g. Reply-To.
type Headers map[string]string

// Attachment represents an email attachment. Attachments are encoded
// using a multipart email with base64 encoding. Use NewAttachment to
// create an Attachment.
type Attachment struct {
	name        string
	contentType string
	data        []byte
}

// NewAttachment returns a new attachment which can be included in the
// Options passed to Send(). If contentType is empty, it defaults
// to application/octet-stream.
func NewAttachment(name string, contentType string, r io.Reader) (*Attachment, error) {
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

// Template is the interface used to send a template as an email.
// There are several types implementing this interface, like
// text/template.Template and html/template.Template.
type Template interface {
	// Execute executes the template with the given data, writing
	// its result to w and returning any potential errors.
	Execute(w io.Writer, data interface{}) error
}

// Options specify several options available when sending an email.
type Options struct {
	// Server to send the email. If empty, the value returned from
	// DefaultServer() is used. See DefaultServer() documentation
	// for the format of this field.
	Server string
	// From address. If empty, defaults to the value from DefaultFrom().
	// Note that this (or DefaultFrom()) always overwrites any From header
	// set using the Header field.
	From string
	// Subject of the email. If non-empty, overwrites any Subject header
	// set using the Headers field.
	Subject string
	// Additional email Headers. Might be nil.
	Headers Headers
	// Attachments to add to the email. See Attachment and NewAttachment.
	Attachments []*Attachment
	// Message indicates the message body. It might be one of the following:
	//
	//  - string
	//  - []byte
	//  - io.Reader
	//  - Template
	Message interface{}
	// Data is only used when Message is a Template. When executing the Template,
	// this is passed as its data argument.
	Data interface{}
}

// Send sends an email to the given addresses and using the given Options. Note that
// if Options is nil, an error is returned. See the Options documentation for
// further information.
func Send(to []string, opts *Options) error {
	if opts == nil {
		return errNoMessage
	}
	from := defaultFrom
	server := defaultServer
	if opts.Server != "" {
		server = opts.Server
	}
	if opts.From != "" {
		from = opts.From
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
	headers := opts.Headers
	if headers == nil {
		headers = make(Headers)
	}
	if opts.Subject != "" {
		headers["Subject"] = opts.Subject
	}
	headers["From"] = from
	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	var message string
	switch x := opts.Message.(type) {
	case Template:
		var b bytes.Buffer
		err := x.Execute(&b, opts.Data)
		if err != nil {
			return fmt.Errorf("error executing message template: %s", err)
		}
		message = b.String()
	case string:
		message = x
	case []byte:
		message = string(x)
	case io.Reader:
		data, err := ioutil.ReadAll(x)
		if err != nil {
			return fmt.Errorf("error reading message body: %s", err)
		}
		message = string(data)
	default:
		return fmt.Errorf("invalid message type %T", opts.Message)
	}
	if len(opts.Attachments) > 0 {
		boundary := "Gondola-Boundary-" + util.RandomString(16)
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		buf.WriteString("--" + boundary + "\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString(message)
		for _, v := range opts.Attachments {
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
	if server == "echo" {
		printer("To: %s\n\n%s\n", strings.Join(to, ", "), buf.String())
		return nil
	}
	return smtp.SendMail(server, auth, from, to, buf.Bytes())
}

// DefaultServer returns the default mail server address.
func DefaultServer() string {
	return defaultServer
}

// SetDefaultServer sets the default mail server adress in the format
// [user:password]@host[:port]. If you want to use CRAM authentication,
// prefix the username with "cram?" - witout quotes, otherwise PLAIN
// authentication is used. Additionally, the special value "echo" can be
// used for testing, and will cause the email to be printed
// to the standard output, rather than sent. The following are valid examples
// of server addresses.
//
//  - localhost
//  - localhost:25
//  - user@gmail.com:patata@smtp.gmail.com
//  - cram?pepe:12345@example.com
//  - echo
//
// The default server value is localhost:25.
func SetDefaultServer(s string) {
	if s == "" {
		s = "localhost:25"
	}
	defaultServer = s
}

// DefaultFrom returns the default From address used
// in outgoing emails. Use gnd.la/config or SetDefaultFrom
// to change it.
func DefaultFrom() string {
	return defaultFrom
}

// SetDefaultFrom sets the default From address used
// in outgoing emails.
func SetDefaultFrom(f string) {
	defaultFrom = f
}

// AdminEmail returns the administrator email
func AdminEmail() string {
	return admin
}

// SetAdminEmail sets the administrator's email. When
// email logging is enabled, errors will be sent to this
// address. You can also change this value using gnd.la/config.
func SetAdminEmail(email string) {
	admin = email
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
