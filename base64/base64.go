// Package base64 implements base64 encoding/decoding
// stripping any = used for padding, thus producing
// invalid base64 but saving a few bytes. It's mainly
// used for encoding values in other parts of gondola
// (like gondola/cookies), but you can feel free to
// use its functions. Just keep in mind that any
// value encoded with Encode() must be decoded
// with Decode(), rather than with the standard
// encoding/base64 package.
package base64

import (
	"encoding/base64"
	"strings"
)

// Encode encodes the given []byte into a base64
// string, using URL encoding and removing any
// = character used for padding.
func Encode(src []byte) string {
	s := base64.URLEncoding.EncodeToString(src)
	// Remove any padding =
	return strings.TrimRight(s, "=")
}

// Decode decodes the given string into a []byte
// using base64 URL encoding. The string does not
// need to be correctly padded.
func Decode(src string) ([]byte, error) {
	// base64 encoded strings must be a multiple of 4
	l := len(src)
	pad := (((l + 3) / 4) * 4) - l
	s := src + strings.Repeat("=", pad)
	return base64.URLEncoding.DecodeString(s)
}
