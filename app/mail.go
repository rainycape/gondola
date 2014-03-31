package app

import (
	"fmt"
	"gnd.la/mail"
	"io"
)

type mailTemplate struct {
	tmpl *tmpl
	ctx  *Context
}

func (t *mailTemplate) Execute(w io.Writer, data interface{}) error {
	return t.tmpl.executeContext(t.ctx, w, "", data, nil)
}

// MailTemplate returns the given template as mail.Template which can be used
// to send App templates with gnd.la/mail.Send. Don't try to use a Template
// as returned by App.LoadTemplate, since functions like reverse or translations
// will not work otherwise because they won't have access to the *Context. Note
// that the lifetime of the returned mail.Template is tied to the lifetime
// of the *Context.
func (c *Context) MailTemplate(template string) (mail.Template, error) {
	t, err := c.app.LoadTemplate(template)
	if err != nil {
		return nil, err
	}
	return &mailTemplate{tmpl: t.(*tmpl), ctx: c}, nil
}

// SendMail is a shorthand function for sending an email from a template.
// The to parameter might be either a []string or a string, in the latter case
// it's parsed with gnd.la/mail.ParseAddressList.
//
// If you need more granularity you can use Context.MailTemplate and gnd.la/mail.Send
// directly. Note that fields other than Message and Data in the opts argument are not
// altered and are passed as is to gnd.la/mail.Send.
func (c *Context) SendMail(to interface{}, template string, data interface{}, opts *mail.Options) error {
	if template != "" {
		t, err := c.MailTemplate(template)
		if err != nil {
			return err
		}
		if opts == nil {
			opts = &mail.Options{}
		}
		opts.Message = t
		opts.Data = data
	}
	var addrs []string
	switch x := to.(type) {
	case string:
		p, err := mail.ParseAddressList(x)
		if err != nil {
			return err
		}
		addrs = p
	case []string:
		addrs = x
	default:
		return fmt.Errorf("invalid to type %T (%v)", to, to)
	}
	return mail.Send(addrs, opts)
}

// MustSendMail works like SendMail, but panics if there's an error.
func (c *Context) MustSendMail(to interface{}, template string, data interface{}, opts *mail.Options) {
	if err := c.SendMail(to, template, data, opts); err != nil {
		panic(err)
	}
}
