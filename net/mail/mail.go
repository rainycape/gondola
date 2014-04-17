// Package mail provides a conveniency interface over net/smtp, to
// facilitate the most common tasks when sending emails.
package mail

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/mail"
	"path"

	"gnd.la/util/generic"
)

const (
	// Emails sent to the Admin pseudo-address will be sent to the
	// administrator (on non-GAE, this means sending it to AdminEmail()).
	Admin = "admin"
)

var (
	defaultServer      = "localhost:25"
	defaultFrom        = ""
	admin              = ""
	errNoMessage       = errors.New("no message specified")
	errNoDestinataries = errors.New("no destinataries specified")
	errNoBody          = errors.New("no message body")
	errNoFrom          = errors.New("missing From: address")
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
// the email.
type Headers map[string]string

// Attachment represents an email attachment.
// See the conveniency function NewAttachment.
type Attachment struct {
	Name        string
	ContentType string
	Data        []byte
	// ContentID is used to reference attachments from
	// the message HTML body. Note that attachments with
	// a non-empty ContentID which is referenced from the
	// HTML will not be included as downloadable attachments.
	// However, if their ContentID is not found in the HTML
	// they will be treated as normal attachments.
	ContentID string
}

// NewAttachment returns a new attachment which can be included in the
// Message passed to Send(). The ContentType is derived from the file name.
func NewAttachment(filename string, r io.Reader) (*Attachment, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if filename == "" {
		filename = "file"
	}
	contentType := mime.TypeByExtension(path.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &Attachment{Name: filename, ContentType: contentType, Data: data}, nil
}

// Message describes an email to be sent. The fields that are mapped
// to headers (e.g From, Subject, To, etc...) overwrite any headers set
// in the Headers field when they're non-empty.
type Message struct {
	// Server to send the email. If empty, the value returned from
	// DefaultServer() is used. See DefaultServer() documentation
	// for the format of this field.
	Server string
	// From address. If empty, defaults to the value from DefaultFrom().
	// Note that this (or DefaultFrom()) always overwrites any From header
	// set using the Headers field.
	From string
	// ReplyTo address.
	ReplyTo string
	// Destinataries fields. These might be either a string or a []string. In the
	// former case, the value is parsed using ParseAddressList.
	// At least of them must be non-empty.
	To, Cc, Bcc interface{}
	// Subject of the email. If non-empty, overwrites any Subject header
	// set using the Headers field.
	Subject string
	// TextBody is the message body to be sent as text/plain. Either this
	// or HTMLBody must be non-empty
	TextBody string
	// HTMLBody is the message body to be sent as text/html. Either this
	// or TextBody must be non-empty
	HTMLBody string
	// Additional email Headers. Might be nil.
	Headers Headers
	// Attachments to add to the email. See Attachment and NewAttachment.
	Attachments []*Attachment
	// Context is used interally by Gondola and should not be altered by users.
	Context interface{}
}

// Send sends an email to the given addresses and using the given Message. Note that
// if Message is nil, an error is returned. See the Options documentation for
// further information.
// This function does not work on App Engine. Use gnd.la/app.Context.SendMail.
func Send(msg *Message) error {
	if msg == nil {
		return errNoMessage
	}
	if msg.TextBody == "" && msg.HTMLBody == "" {
		return errNoBody
	}
	to, err := parseDestinataries(msg.To, "To")
	if err != nil {
		return err
	}
	cc, err := parseDestinataries(msg.Cc, "Cc")
	if err != nil {
		return err
	}
	bcc, err := parseDestinataries(msg.Bcc, "Bcc")
	if err != nil {
		return err
	}
	if len(to) == 0 && len(cc) == 0 && len(bcc) == 0 {
		return errNoDestinataries
	}
	return sendMail(to, cc, bcc, msg)
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

func parseDestinataries(value interface{}, name string) ([]string, error) {
	var addrs []string
	var err error
	switch x := value.(type) {
	case []string:
		addrs = x
	case string:
		if x == Admin {
			addrs = []string{x}
			break
		}
		addrs, err = ParseAddressList(x)
		if err != nil {
			err = fmt.Errorf("invalid %s field: %s", name, err)
		}
	case nil:
	default:
		err = fmt.Errorf("invalid %s field type %T", name, value)
	}
	return addrs, err
}
