package facebook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func decodeBase64(s string) ([]byte, error) {
	// Facebook has the ugly hobby of trimming '=' from
	// the end of base64 encoded values.
	if mod := len(s) % 4; mod != 0 {
		s += strings.Repeat("=", 4-mod)
	}
	return base64.URLEncoding.DecodeString(s)
}

func ParseSignedRequest(app *App, req string) (map[string]interface{}, error) {
	fields := strings.SplitN(req, ".", 2)
	signature, err := decodeBase64(fields[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding facebook signature: %s", err)
	}
	mac := hmac.New(sha256.New, []byte(app.Secret))
	// The signature refers to the encoded base64 string, not
	// to the initial data.
	if _, err := io.WriteString(mac, fields[1]); err != nil {
		return nil, err
	}
	result := mac.Sum(nil)
	if !hmac.Equal(result, signature) {
		return nil, fmt.Errorf("bad facebook request signature")
	}
	payload, err := decodeBase64(fields[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding facebook payload: %s", err)
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	var m map[string]interface{}
	err = dec.Decode(&m)
	return m, err
}
