package oauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
)

// Base returns the signature base string for the given input parameters.
func Base(method string, url string, values url.Values) string {
	return fmt.Sprintf("%s&%s&%s", method, encode(url), encodePlusEncoded(values.Encode()))
}

// Sign returns the HMAC-SHA1 signature with the given signature base and
// secrets.
func Sign(base string, clientSecret string, tokenSecret string) string {
	key := encode(clientSecret) + "&" + encode(tokenSecret)
	h := hmac.New(sha1.New, []byte(key))
	io.WriteString(h, base)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
