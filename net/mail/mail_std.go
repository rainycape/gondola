// +build !appengine

package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"sort"

	"gnd.la/util/stringutil"
)

var (
	errNoAdminEmail = errors.New("mail.Admin specified as a mail destinary, but no mail.AdminEmail() has been set")
	crlf            = []byte("\r\n")
)

func joinAddrs(addrs []string) (string, error) {
	var values []string
	for _, v := range addrs {
		if v == Admin {
			addr := AdminEmail()
			if addr == "" {
				return "", errNoAdminEmail
			}
			addrs, err := ParseAddressList(addr)
			if err != nil {
				return "", fmt.Errorf("invalid admin address %q: %s", addr, err)
			}
			values = append(values, addrs...)
			continue
		}
		values = append(values, v)
	}
	return strings.Join(values, ", "), nil
}

func makeBoundary() string {
	return "Gondola-Boundary-" + stringutil.Random(32)
}

func sendMail(to []string, cc []string, bcc []string, msg *Message) error {
	from := Config.DefaultFrom
	server := Config.MailServer
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
	if msg.ReplyTo != "" {
		headers["Reply-To"] = msg.ReplyTo
	}
	var err error
	if len(to) > 0 {
		headers["To"], err = joinAddrs(to)
		if err != nil {
			return err
		}
		for ii, v := range to {
			if v == Admin {
				if Config.AdminEmail == "" {
					return errNoAdminEmail
				}
				to[ii] = Config.AdminEmail
			}
		}
	}
	if len(cc) > 0 {
		headers["Cc"], err = joinAddrs(cc)
		if err != nil {
			return err
		}
	}
	if len(bcc) > 0 {
		headers["Bcc"], err = joinAddrs(bcc)
		if err != nil {
			return err
		}
	}
	hk := make([]string, 0, len(headers))
	for k := range headers {
		hk = append(hk, k)
	}
	sort.Strings(hk)
	for _, k := range hk {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, headers[k]))
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	mw := multipart.NewWriter(&buf)
	mw.SetBoundary(makeBoundary())
	// Create a multipart mixed first
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed;\r\n\tboundary=%q\r\n\r\n", mw.Boundary())
	var bodyWriter *multipart.Writer
	if msg.TextBody != "" && msg.HTMLBody != "" {
		boundary := makeBoundary()
		outerHeader := make(textproto.MIMEHeader)
		// First part is a multipart/alternative, which contains the text and html bodies
		outerHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", boundary))
		iw, err := mw.CreatePart(outerHeader)
		if err != nil {
			return err
		}
		bodyWriter = multipart.NewWriter(iw)
		bodyWriter.SetBoundary(boundary)
	} else {
		bodyWriter = mw
	}
	if msg.TextBody != "" {
		textHeader := make(textproto.MIMEHeader)
		textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
		tpw, err := bodyWriter.CreatePart(textHeader)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(tpw, msg.TextBody); err != nil {
			return err
		}
		tpw.Write(crlf)
		tpw.Write(crlf)
	}
	attached := make(map[*Attachment]bool)
	if msg.HTMLBody != "" {
		var htmlAttachments []*Attachment
		for _, v := range msg.Attachments {
			if v.ContentID != "" && strings.Contains(msg.HTMLBody, fmt.Sprintf("cid:%s", v.ContentID)) {
				htmlAttachments = append(htmlAttachments, v)
				attached[v] = true
			}
		}
		var htmlWriter *multipart.Writer
		if len(htmlAttachments) > 0 {
			relatedHeader := make(textproto.MIMEHeader)
			relatedBoundary := makeBoundary()
			relatedHeader.Set("Content-Type", fmt.Sprintf("multipart/related; boundary=%q; type=\"text/html\"", relatedBoundary))
			rw, err := bodyWriter.CreatePart(relatedHeader)
			if err != nil {
				return err
			}
			htmlWriter = multipart.NewWriter(rw)
			htmlWriter.SetBoundary(relatedBoundary)
		} else {
			htmlWriter = bodyWriter
		}
		htmlHeader := make(textproto.MIMEHeader)
		htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
		thw, err := htmlWriter.CreatePart(htmlHeader)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(thw, msg.HTMLBody); err != nil {
			return err
		}
		thw.Write(crlf)
		thw.Write(crlf)
		for _, v := range htmlAttachments {
			attachmentHeader := make(textproto.MIMEHeader)
			attachmentHeader.Set("Content-Type", v.ContentType)
			attachmentHeader.Set("Content-Disposition", "inline")
			attachmentHeader.Set("Content-ID", fmt.Sprintf("<%s>", v.ContentID))
			attachmentHeader.Set("Content-Transfer-Encoding", "base64")
			aw, err := htmlWriter.CreatePart(attachmentHeader)
			if err != nil {
				return err
			}
			b := make([]byte, base64.StdEncoding.EncodedLen(len(v.Data)))
			base64.StdEncoding.Encode(b, v.Data)
			aw.Write(b)
		}
		if htmlWriter != bodyWriter {
			if err := htmlWriter.Close(); err != nil {
				return err
			}
		}
	}
	if bodyWriter != mw {
		if err := bodyWriter.Close(); err != nil {
			return err
		}
	}
	for _, v := range msg.Attachments {
		if attached[v] {
			continue
		}
		attachmentHeader := make(textproto.MIMEHeader)
		attachmentHeader.Set("Content-Type", v.ContentType)
		attachmentHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", v.Name))
		attachmentHeader.Set("Content-Transfer-Encoding", "base64")
		aw, err := mw.CreatePart(attachmentHeader)
		if err != nil {
			return err
		}
		b := make([]byte, base64.StdEncoding.EncodedLen(len(v.Data)))
		base64.StdEncoding.Encode(b, v.Data)
		aw.Write(b)
	}
	if err := mw.Close(); err != nil {
		return err
	}
	if server == "echo" {
		printer(buf.String())
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
