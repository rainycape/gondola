// Package mail provides a conveniency interface over net/smtp, to
// facilitate the most common tasks when sending emails.
package mail

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/mail"

	"gnd.la/util/generic"
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
// Message passed to Send(). If contentType is empty, it defaults
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

// Message describes an email to be sent.
type Message struct {
	// Server to send the email. If empty, the value returned from
	// DefaultServer() is used. See DefaultServer() documentation
	// for the format of this field.
	Server string
	// From address. If empty, defaults to the value from DefaultFrom().
	// Note that this (or DefaultFrom()) always overwrites any From header
	// set using the Headers field.
	From string
	// Subject of the email. If non-empty, overwrites any Subject header
	// set using the Headers field.
	Subject string
	// Additional email Headers. Might be nil.
	Headers Headers
	// Attachments to add to the email. See Attachment and NewAttachment.
	Attachments []*Attachment
	// Body indicates the message body. It might be one of the following:
	//
	//  - string
	//  - []byte
	//  - io.Reader
	//  - Template
	Body interface{}
	// Data is only used when Body is a Template. When executing the Template,
	// this is passed as its data argument.
	Data interface{}
	// HTML indicates if the message body is HTML.
	HTML bool
	// Context is used interally by Gondola and should not be altered by users.
	Context interface{}
}

// Send sends an email to the given addresses and using the given Message. Note that
// if Message is nil, an error is returned. See the Options documentation for
// further information.
// This function does not work on App Engine. Use gnd.la/app.Context.SendMail.
func Send(to []string, msg *Message) error {
	if msg == nil {
		return errNoMessage
	}
	return sendMail(to, msg)
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

func messageBody(msg *Message) (string, error) {
	var body string
	switch x := msg.Body.(type) {
	case Template:
		var b bytes.Buffer
		err := x.Execute(&b, msg.Data)
		if err != nil {
			return "", fmt.Errorf("error executing message template: %s", err)
		}
		body = b.String()
	case string:
		body = x
	case []byte:
		body = string(x)
	case io.Reader:
		data, err := ioutil.ReadAll(x)
		if err != nil {
			return "", fmt.Errorf("error reading message body: %s", err)
		}
		body = string(data)
	case nil:
	default:
		return "", fmt.Errorf("invalid body type %T", msg.Body)
	}
	return body, nil
}
