package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	oauth "github.com/fiam/goauth"
	"gnd.la/util/textutil"
	"net/http"
)

const (
	REQUEST_TOKEN_URL = "http://twitter.com/oauth/request_token"
	ACCESS_TOKEN_URL  = "http://twitter.com/oauth/access_token"
	AUTHORIZATION_URL = "http://twitter.com/oauth/authorize"
	STATUS_URL        = "http://api.twitter.com/1.1/statuses/update.json"
)

var (
	errNoApp   = errors.New("missing app")
	errNoToken = errors.New("missing token")
)

func parse(raw string, key, secret *string) error {
	fields, err := textutil.SplitFields(raw, ":", "'\"")
	if err != nil {
		return err
	}
	if len(fields) != 2 {
		return fmt.Errorf("invalid number of fields %d, must have 2", len(fields))
	}
	*key = fields[0]
	*secret = fields[1]
	return nil
}

// App represents a Twitter application, with its
// consumer key and secret. To register an application
// go to https://dev.twitter.com
type App struct {
	Key    string
	Secret string
}

// Parse parses the app credentials from its string representation.
// It must have the form key:secret. If key or secret contain the
// ':' character you must quote them (e.g. "k:ey":"sec:ret").
func (a *App) Parse(s string) error {
	if err := parse(s, &a.Key, &a.Secret); err != nil {
		return fmt.Errorf("error parsing twitter app: %s", err)
	}
	return nil
}

// Token represents a key/secret pair for accessing Twitter
// services on behalf on another user.
type Token struct {
	Key    string
	Secret string
}

// Parse parses the token fields from its string representation.
// It must have the form key:secret. If key or secret contain the
// ':' character you must quote them (e.g. "k:ey":"sec:ret").
func (t *Token) Parse(s string) error {
	if err := parse(s, &t.Key, &t.Secret); err != nil {
		return fmt.Errorf("error parsing twitter token: %s", err)
	}
	return nil
}

type TwitterError struct {
	Message    string
	Code       int
	StatusCode int
}

func (t *TwitterError) Error() string {
	return fmt.Sprintf("%s (code %d, status code %d)", t.Message, t.Code, t.StatusCode)
}

func post(app *App, token *Token, url string, data map[string]string) (*http.Response, error) {
	if app == nil {
		return nil, errNoApp
	}
	if token == nil {
		return nil, errNoToken
	}
	conn := &oauth.OAuthConsumer{
		Service:          "twitter",
		RequestTokenURL:  REQUEST_TOKEN_URL,
		AccessTokenURL:   ACCESS_TOKEN_URL,
		AuthorizationURL: AUTHORIZATION_URL,
		ConsumerKey:      app.Key,
		ConsumerSecret:   app.Secret,
		CallBackURL:      "oob",
	}
	params := oauth.Params{}
	for k, v := range data {
		params = append(params, &oauth.Pair{Key: k, Value: v})
	}
	return conn.Post(STATUS_URL, params, &oauth.AccessToken{
		Token:  token.Key,
		Secret: token.Secret,
	})
}

type twitterError struct {
	Message string
	Code    int
}

type twitterErrors struct {
	Errors []twitterError
}

func parseTwitterResponse(resp *http.Response, out interface{}) error {
	if resp.StatusCode != http.StatusOK {
		var message string
		var code int
		var errs twitterErrors
		dec := json.NewDecoder(resp.Body)
		if dec.Decode(&errs) == nil && len(errs.Errors) > 0 {
			message = errs.Errors[0].Message
			code = errs.Errors[0].Code
		}
		return &TwitterError{
			Message:    message,
			Code:       code,
			StatusCode: resp.StatusCode,
		}
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}
