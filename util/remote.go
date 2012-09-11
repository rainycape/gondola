package util

import (
	"net/http"
	"strings"
)

var (
	TrustXHeaders = true
)

func RemoteAddress(r *http.Request) string {
	remote := ""
	if TrustXHeaders {
		remote = r.Header.Get("X-Real-IP")
	}
	if remote == "" {
		remote = r.RemoteAddr
	}
	if strings.Contains(remote, ":") {
		remote = strings.Split(remote, ":")[0]
	}
	return remote
}
