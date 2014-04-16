// +build appengine

package mail

import (
	"errors"
	nmail "net/mail"

	"appengine"
	"appengine/mail"
)

var (
	errNoContext   = errors.New("no App Engine context available - if you're using gnd.la/mail.Send, use gnd.la/app.Context.SendMail instead")
	allowedHeaders = []string{
		"In-Reply-To",
		"List-Id",
		"List-Unsubscribe",
		"On-Behalf-Of",
		"References",
		"Resent-Date",
		"Resent-From",
		"Resent-To",
	}
)

func sendMail(to []string, msg *Message) error {
	c, ok := msg.Context.(appengine.Context)
	if !ok {
		return errNoContext
	}
	body, err := messageBody(msg)
	if err != nil {
		return err
	}
	gaeMsg := &mail.Message{
		Sender:  msg.From,
		ReplyTo: msg.Headers["Reply-To"],
		To:      to,
		Subject: msg.Subject,
		Headers: make(nmail.Header),
	}
	if cc := msg.Headers["Cc"]; cc != "" {
		addrs, err := ParseAddressList(cc)
		if err != nil {
			return err
		}
		gaeMsg.Cc = addrs
	}
	if bcc := msg.Headers["Bcc"]; bcc != "" {
		addrs, err := ParseAddressList(bcc)
		if err != nil {
			return err
		}
		gaeMsg.Bcc = addrs
	}
	if msg.HTML {
		gaeMsg.HTMLBody = body
	} else {
		gaeMsg.Body = body
	}
	for _, v := range msg.Attachments {
		gaeMsg.Attachments = append(gaeMsg.Attachments, mail.Attachment{
			Name:      v.name,
			Data:      v.data,
			ContentID: v.contentType,
		})
	}
	for _, v := range allowedHeaders {
		if value := msg.Headers[v]; value != "" {
			gaeMsg.Headers[v] = append(gaeMsg.Headers[v], value)
		}
	}
	return mail.Send(c, gaeMsg)
}
