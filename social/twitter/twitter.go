package twitter

import (
	"errors"
	"fmt"
	"net/url"

	"gnd.la/net/httpclient"
	"gnd.la/net/oauth"
	"gnd.la/util/stringutil"
)

const (
	REQUEST_TOKEN_URL  = "https://twitter.com/oauth/request_token"
	ACCESS_TOKEN_URL   = "https://twitter.com/oauth/access_token"
	AUTHORIZATION_URL  = "https://twitter.com/oauth/authorize"
	AUTHENTICATION_URL = "https://twitter.com/oauth/authenticate"
	API_BASE_URL       = "https://api.twitter.com/1.1/"
)

var (
	statusPath = "statuses/update.json"
	verifyPath = "account/verify_credentials.json"
	STATUS_URL = API_BASE_URL + "statuses/update.json"
	errNoApp   = errors.New("missing app")
	errNoToken = errors.New("missing token")
)

func parse(raw string, key, secret *string) error {
	fields, err := stringutil.SplitFieldsOptions(raw, ":", &stringutil.SplitOptions{ExactCount: 2})
	if err != nil {
		return err
	}
	*key = fields[0]
	*secret = fields[1]
	return nil
}

// App represents a Twitter application, with its
// consumer key and secret. To register an application
// go to https://dev.twitter.com
type App struct {
	Key        string
	Secret     string
	Client     *httpclient.Client
	httpClient *httpclient.Client
}

func (a *App) Clone(ctx httpclient.Context) *App {
	ac := *a
	ac.Client = ac.Client.Clone(ctx)
	return &ac
}

func (a *App) client() *httpclient.Client {
	if a.Client != nil {
		return a.Client
	}
	if a.httpClient == nil {
		a.httpClient = httpclient.New(nil)
	}
	return a.httpClient
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

func newConsumer(app *App) *oauth.Consumer {
	return &oauth.Consumer{
		Key:              app.Key,
		Secret:           app.Secret,
		Service:          "twitter",
		RequestTokenURL:  REQUEST_TOKEN_URL,
		AccessTokenURL:   ACCESS_TOKEN_URL,
		AuthorizationURL: AUTHORIZATION_URL,
		CallbackURL:      "oob",
		Client:           app.client(),
	}
}

func sendReq(app *App, token *Token, method string, path string, data map[string]string, out interface{}) error {
	if app == nil {
		return errNoApp
	}
	if token == nil {
		return errNoToken
	}
	c := newConsumer(app)
	resp, err := c.SendRequest(method, API_BASE_URL+path, asValues(data), &oauth.Token{
		Key:    token.Key,
		Secret: token.Secret,
	})
	if err != nil {
		return err
	}
	return parseTwitterResponse(resp, out)
}

type twitterError struct {
	Message string
	Code    int
}

type twitterErrors struct {
	Errors []twitterError
}

func parseTwitterResponse(resp *httpclient.Response, out interface{}) error {
	if resp.IsOK() {
		var message string
		var code int
		var errs twitterErrors
		if resp.JSONDecode(&errs) == nil && len(errs.Errors) > 0 {
			message = errs.Errors[0].Message
			code = errs.Errors[0].Code
		}
		return &TwitterError{
			Message:    message,
			Code:       code,
			StatusCode: resp.StatusCode,
		}
	}
	if err := resp.JSONDecode(out); err != nil {
		return err
	}
	return nil
}

func asValues(data map[string]string) url.Values {
	values := make(url.Values)
	for k, v := range data {
		values.Add(k, v)
	}
	return values
}
