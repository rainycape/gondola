package oauth

import (
	"fmt"
	"net/url"

	"gnd.la/net/httpclient"
)

type Token struct {
	Key    string
	Secret string
}

func parseToken(resp *httpclient.Response) (*Token, error) {
	defer resp.Close()
	b, err := resp.ReadAll()
	if err != nil {
		return nil, err
	}
	s := string(b)
	if !resp.IsOK() {
		return nil, fmt.Errorf("oAuth service returned non-200 status code %d: %s", resp.StatusCode, s)
	}
	values, err := url.ParseQuery(s)
	if err != nil {
		return nil, err
	}
	key := values.Get("oauth_token")
	secret := values.Get("oauth_token_secret")
	if key == "" || secret == "" {
		return nil, fmt.Errorf("can't parse token from %q", s)
	}
	return &Token{
		Key:    key,
		Secret: secret,
	}, nil
}
