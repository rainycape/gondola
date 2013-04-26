// Package mail provides a conviency interface over net/smtp, to
// facilitate the most common tasks.
package mail

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"
)

var defaultServer = "localhost:25"

// SendVia sends an email using the specified server from the specified address
// to the given addresses (separated by commmas). Addittional headers might
// be specified, like Subject or Reply-To. To include authentication info,
// embed it into the server address (e.g. user@gmail.com:patata@smtp.gmail.com).
// If you want to use CRAM authentication, prefix the username with cram?
// (e.g. cram?pepe:12345@example.com), otherwise PLAIN is used.
// If server is empty, it defaults to localhost:25
func SendVia(server, from, to, message string, headers map[string]string) error {
	if server == "" {
		server = "localhost:25"
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
	for k, v := range headers {
		buf.Write([]byte(fmt.Sprintf("%s: %s\r\n", k, v)))
	}
	buf.Write([]byte{'\r', '\n'})
	buf.Write([]byte(message))
	return smtp.SendMail(server, auth, from, strings.Split(to, ","), buf.Bytes())
}

// Send works like SendVia(), but uses the mail server
// specified by gondola/defaults/MailServer.
func Send(from, to, message string, headers map[string]string) error {
	return SendVia(defaultServer, from, to, message, headers)
}

// SetDefaultServer sets the default mail server. This function
// exists only to avoid an import cycle and you should not call it.
// Instead, use gondola/defaults/SetMailServer
func SetDefaultServer(s string) {
	defaultServer = s
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
