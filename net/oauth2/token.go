package oauth2

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gnd.la/net/httpclient"
)

// TokenType indicates the type of oAuth 2 token.
// At this time, only bearer tokens are supported.
type TokenType int

const (
	// TokenTypeBearer represents a token of type bearer.
	TokenTypeBearer TokenType = iota
)

// Token represents an oAuth 2 token. Note that
// not all oAuth 2 providers use all the fields.
type Token struct {
	// Key is the token value.
	Key string
	// Scopes contains the scopes granted by the user. Note that
	// not all providers return this information.
	Scopes []string
	// Refresh is used by Google tokens to obtain a new fresh
	// token from an expired one.
	Refresh string
	// Type is the token type. Currently, this is always
	// TokenTypeBearer.
	Type TokenType
	// Expires indicates when the token expires. If this field is
	// the zero time, it indicates that the provider did not provide
	// an expiration. If the expiration time returned by the provider
	// is the string "0" (as e.g. Facebook does), it's interpreted as
	// a non-expiring token and its expiration is set 100 years into
	// the future.
	Expires time.Time
}

// ParseToken parses an oAuth 2 token from a query string.
// See also NewToken and ParseJSONToken.
func ParseToken(r io.Reader) (*Token, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}

	tokenType := values.Get("token_type")
	if tokenType != "" && strings.ToLower(tokenType) != "bearer" {
		return nil, fmt.Errorf("unknown token type %q", tokenType)
	}
	key := values.Get("access_token")
	if key == "" {
		return nil, fmt.Errorf("missing access_token from response %q", string(data))
	}
	var scopes []string
	scope := values.Get("scope")
	if scope != "" {
		scopes = strings.Split(scope, ",")
	}
	var expires time.Time
	expiresIn := values.Get("expires")
	if expiresIn == "" {
		expiresIn = values.Get("expires_in")
	}
	if expiresIn != "" {
		var duration time.Duration
		// Keep the expires as the zero time if expires
		// and expires_in are not specified. This allows the Facebook
		// code to detect when a token was already extended.
		if expiresIn == "0" {
			// Returned by FB for tokens that not expire, set it
			// 100 years, since I don't think it will be my problem
			// if this ends up breaking some code.
			duration = time.Hour * 24 * 365 * 100
		} else {
			seconds, err := strconv.ParseUint(expiresIn, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid expires %q: %s", expiresIn, err)
			}
			duration = time.Duration(seconds) * time.Second
		}
		expires = time.Now().UTC().Add(duration)
	}
	return &Token{
		Key:     key,
		Scopes:  scopes,
		Type:    TokenTypeBearer,
		Refresh: values.Get("refresh_token"),
		Expires: expires,
	}, nil
}

// ParseJSONToken parses an oAuth 2 token from its JSON
// representation. See also ParseToken.
func ParseJSONToken(r io.Reader) (*Token, error) {
	dec := json.NewDecoder(r)
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("error decoding JSON token: %s", err)
	}
	tokenType, _ := m["token_type"].(string)
	if tokenType != "" && strings.ToLower(tokenType) != "bearer" {
		return nil, fmt.Errorf("unknown token type %q", tokenType)
	}
	key, _ := m["access_token"].(string)
	if key == "" {
		return nil, fmt.Errorf("access_token missing from JSON response %v", m)
	}
	var expires time.Time
	if expiresIn, ok := m["expires_in"].(float64); ok {
		expires = time.Now().UTC().Add(time.Second * time.Duration(expiresIn))
	}
	refresh, _ := m["refresh_token"].(string)
	return &Token{
		Key:     key,
		Type:    TokenTypeBearer,
		Refresh: refresh,
		Expires: expires,
	}, nil
}

// NewToken returns a Token from an httpclient.Response. Note that
// this function supports both JSON encoded tokens and query string
// encoded ones. It uses the response Content-Type to decide which
// parsing strategy to use.
func NewToken(r *httpclient.Response) (*Token, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		return ParseJSONToken(r.Body)
	}
	return ParseToken(r.Body)
}
