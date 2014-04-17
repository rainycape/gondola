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

func prepareAddrs(addrs []string, toAdmins *bool) []string {
	var ret []string
	for _, v := range addrs {
		if v == Admin {
			*toAdmins = true
			continue
		}
		ret = append(ret, v)
	}
	return ret
}

func sendMail(to []string, cc []string, bcc []string, msg *Message) error {
	c, ok := msg.Context.(appengine.Context)
	if !ok {
		return errNoContext
	}
	var toAdmins bool
	gaeMsg := &mail.Message{
		Sender:   msg.From,
		ReplyTo:  msg.ReplyTo,
		To:       prepareAddrs(to, &toAdmins),
		Cc:       prepareAddrs(cc, &toAdmins),
		Bcc:      prepareAddrs(bcc, &toAdmins),
		Subject:  msg.Subject,
		Body:     msg.TextBody,
		HTMLBody: msg.HTMLBody,
		Headers:  make(nmail.Header),
	}
	for _, v := range msg.Attachments {
		gaeMsg.Attachments = append(gaeMsg.Attachments, mail.Attachment{
			Name:      v.Name,
			Data:      v.Data,
			ContentID: v.ContentID,
		})
	}
	for _, v := range allowedHeaders {
		if value := msg.Headers[v]; value != "" {
			gaeMsg.Headers[v] = append(gaeMsg.Headers[v], value)
		}
	}
	if toAdmins {
		if err := mail.SendToAdmins(c, gaeMsg); err != nil {
			return nil
		}
	}
	return mail.Send(c, gaeMsg)
}
