package util

import (
	"net/http"
	"net/url"
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

func From(r *http.Request) string {
	from := r.FormValue("from")
	if from == "" {
		ref := r.Referer()
		if ref != "" {
			refURL, err := url.Parse(ref)
			if err == nil {
				if r.Host == refURL.Host {
					from = ref
				}
			}
		}
	}
	return from
}
