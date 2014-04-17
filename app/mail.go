package app

import (
	"bytes"
	"strings"

	"gnd.la/net/mail"
)

func (c *Context) mailTemplate(template string) (*linkedTemplate, error) {
	t, err := c.app.LoadTemplate(template)
	if err != nil {
		return nil, err
	}
	return &linkedTemplate{tmpl: t.(*tmpl), ctx: c}, nil
}

// SendMail is a shorthand function for sending an email from a template.
// If the loaded gnd.la/template.Template.ContentType() returns a string
// containing "html", the gnd.la/net/mail.Message HTMLBody field is set, other
// the TextBody field is used. Note that if template is empty, the msg is
// passed unmodified to mail.Send(). Other Message fields are never altered.
//
// Note: mail.Send does not work on App Engine, users must always use this function instead.
func (c *Context) SendMail(template string, data interface{}, msg *mail.Message) error {
	if template != "" {
		t, err := c.mailTemplate(template)
		if err != nil {
			return err
		}
		if msg == nil {
			msg = &mail.Message{}
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return err
		}
		if strings.Contains(t.tmpl.tmpl.ContentType(), "/html") {
			msg.HTMLBody = buf.String()
		} else {
			msg.TextBody = buf.String()
		}
	}
	c.prepareMessage(msg)
	return mail.Send(msg)
}

// MustSendMail works like SendMail, but panics if there's an error.
func (c *Context) MustSendMail(template string, data interface{}, msg *mail.Message) {
	if err := c.SendMail(template, data, msg); err != nil {
		panic(err)
	}
}
