package oauth

import (
	"bytes"
	"fmt"
)

var hex = "0123456789ABCDEF"

// encode percent-encodes a string as defined in RFC 3986.
func encode(s string) string {
	var buf bytes.Buffer
	for _, c := range []byte(s) {
		if isEncodable(c) {
			if c == '+' {
				// replace plus-encoding with percent-encoding
				buf.WriteString("%2520")
			} else {
				buf.WriteByte('%')
				buf.WriteByte(hex[c>>4])
				buf.WriteByte(hex[c&15])
			}
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func encodeQuoted(key, value string) string {
	return fmt.Sprintf("%s=\"%s\"", encode(key), encode(value))
}

// isEncodable returns true if a given character should be percent-encoded
// according to RFC 3986.
func isEncodable(c byte) bool {
	// return false if c is an unreserved character (see RFC 3986 section 2.3)
	switch {
	case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'):
		return false
	case c >= '0' && c <= '9':
		return false
	case c == '-' || c == '.' || c == '_' || c == '~':
		return false
	}
	return true
}
