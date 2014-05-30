// Package urlutil contains utility functions
// related to URLs.
package urlutil

import (
	"net/url"
	"strings"
)

// isAbs returns true of the URL is absolute.
// This function considers protocol-relative
// URLs to be absolute
func isAbs(u *url.URL) bool {
	return u.IsAbs() || strings.HasPrefix(u.String(), "//")
}

// SameHost returns true iff both URLs point
// to the same host. It works for both absolute
// and relative URLs.
func SameHost(url1, url2 string) bool {
	if url1 == "" || url2 == "" {
		return true
	}
	u1, err := url.Parse(url1)
	if err != nil {
		return false
	}
	if !isAbs(u1) {
		return true
	}
	u2, err := url.Parse(url2)
	if err != nil {
		return false
	}
	if !isAbs(u2) {
		return true
	}
	return u1.Host == u2.Host
}

// Join returns the result of joining the base URL
// with the rel URL. If either base or rel are not
// valid URLs, an error will be returned.
func Join(base string, rel string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := url.Parse(rel)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(r).String(), nil
}

// MustJoin works like Join, but panics if there's
// an error.
func MustJoin(base string, rel string) string {
	u, err := Join(base, rel)
	if err != nil {
		panic(err)
	}
	return u
}

// IsURL returns true iff s looks like a URL.
func IsURL(s string) bool {
	return strings.Contains(s, "://") || strings.HasPrefix(s, "//")
}

// AppendQuery appends the given query string as an url.Values to the given
// URL. It works correctly even if the URL already has a query string.
func AppendQuery(s string, query url.Values) string {
	sep := "?"
	if strings.IndexByte(s, '?') >= 0 {
		sep = "&"
	}
	return s + sep + query.Encode()
}
