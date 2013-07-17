package util

import (
	"net/url"
)

// EqualHosts returns true iff both URLs point
// to the same host. It works for both absolute
// and relative URLs.
func EqualHosts(url1, url2 string) bool {
	if url1 == "" || url2 == "" {
		return true
	}
	u1, err := url.Parse(url1)
	if err != nil {
		return false
	}
	if !u1.IsAbs() {
		return true
	}
	u2, err := url.Parse(url2)
	if err != nil {
		return false
	}
	if !u2.IsAbs() {
		return true
	}
	return u1.Host == u2.Host
}
