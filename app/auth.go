package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	errNoAuthorizationHeader = errors.New("no Authorization header sent")
)

// parseUserinfo parses a string with username and password
// separated by a ':' and returns a Userinfo. Note that
// the characters ':' in the username and password
// should be escaped, otherwise an error will be returned.
func parseUserinfo(val string) (*url.Userinfo, error) {
	parts := strings.Split(val, ":")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid Userinfo: %q", val)
	}
	u, err := url.QueryUnescape(parts[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 1 {
		return url.User(u), nil
	}
	p, err := url.QueryUnescape(parts[1])
	if err != nil {
		return nil, err
	}
	return url.UserPassword(u, p), nil
}

// BasicAuth returns the basic user authentication info as a *url.Userinfo.
// If the Authorization is not present or it's not correctly formed, an
// error is returned.
func (c *Context) BasicAuth() (*url.Userinfo, error) {
	var authorization string
	if c.R != nil {
		authorization = c.R.Header.Get("Authorization")
	}
	if authorization == "" {
		return nil, errNoAuthorizationHeader
	}
	fields := strings.SplitN(authorization, " ", 2)
	if len(fields) != 2 || fields[0] != "Basic" {
		return nil, fmt.Errorf("invalid Authorization header %q", authorization)
	}

	decoded, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return nil, err
	}
	return parseUserinfo(string(decoded))
}
