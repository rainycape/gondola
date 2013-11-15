package oauth

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Token struct {
	Key    string
	Secret string
}

func parseToken(resp *http.Response) (*Token, error) {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	s := string(b)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oAuth service returned non-200 status code %d: s", resp.StatusCode, s)
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
