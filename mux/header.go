package mux

import (
	"net/http"
)

// A Header represents the key-value pairs in an HTTP header.
// Header is also convertible to http.Header.
type Header http.Header

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h Header) Add(key, value string) {
	http.Header(h).Add(key, value)
}

// Set sets the header entries associated with key to
// the single element value.  It replaces any existing
// values associated with key.
func (h Header) Set(key, value string) {
	http.Header(h).Set(key, value)
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns "".
// To access multiple values of a key, access the map directly
// with CanonicalHeaderKey.
func (h Header) Get(key string) string {
	return http.Header(h).Get(key)
}

// Del deletes the values associated with key.
func (h Header) Del(key string) {
	http.Header(h).Del(key)
}

// CanonicalHeaderKey returns the canonical format of the
// header key s.  The canonicalization converts the first
// letter and any letter following a hyphen to upper case;
// the rest are converted to lowercase.  For example, the
// canonical key for "accept-encoding" is "Accept-Encoding".
func CanonicalHeaderKey(s string) string {
	return http.CanonicalHeaderKey(s)
}
