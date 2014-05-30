package oauth2

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gnd.la/net/httpclient"
	"gnd.la/net/urlutil"
	"gnd.la/util/stringutil"
)

// Client represents an oAuth 2 client. Use New
// to initialize a *Client.
type Client struct {
	// Id is the application ID, obtained from the oAuth 2 provider.
	Id string
	// Secret is the application secret, obtained from the oAuth 2
	// provider.
	Secret string
	// AuthorizationURL is the URL for redirecting a user to perform client
	// authorization.
	AuthorizationURL string
	// AuthorizationParameters lists additional parameters
	// to be sent to AuthorizationURL when requesting authorization
	// from a user.
	AuthorizationParameters map[string]string
	// ExchangeURL is the URL for exchanging an oAuth 2
	// code for a token.
	ExchangeURL string
	// ExchangeParameters lists additional parameters to be sent
	// to ExchangeURL when exchanging a code for a token.
	ExchangeParameters map[string]string
	// HTTPClient is the underlying HTTP client used by this
	// oAuth 2 Client. It doesn't need to be explicitely
	// initialized, it will be automatically set up on the
	// first HTTP request.
	HTTPClient *httpclient.Client
	// ResponseHasError is used to check if an *httpclient.Response
	// from the provider represents an error. If this field is
	// nil, responses with non 2xx codes will be considered errors.
	ResponseHasError func(*httpclient.Response) bool
	// DecodeError is used to decode an error from an *httpclient.Response
	// which has been determined to contain an error. If this field is
	// nil, the entire response body will be returned as an error using
	// errors.New.
	DecodeError func(*httpclient.Response) error
	// ScopeSeparator indicates the string used to separate scopes. If
	// empty, it defaults to ",". Note that some provides use ","
	// (e.g. Facebook), while others use an space " " (e.g. Google).
	ScopeSeparator string
}

// New returns a new oAuth 2 Client. The authorization parameter
// must be the URL for redirecting a user to perform client
// authorization. Exchange is the URL for exchanging an oAuth 2
// code for a token.
func New(authorization string, exchange string) *Client {
	return &Client{
		AuthorizationURL: authorization,
		ExchangeURL:      exchange,
		HTTPClient:       httpclient.New(nil),
	}
}

func (c *Client) client() *httpclient.Client {
	if c.HTTPClient == nil {
		c.HTTPClient = httpclient.New(nil)
	}
	return c.HTTPClient
}

func (c *Client) responseHasError(r *httpclient.Response) bool {
	if c.ResponseHasError != nil {
		return c.ResponseHasError(r)
	}
	return !r.IsOK()
}

func (c *Client) decodeError(r *httpclient.Response) error {
	if c.DecodeError != nil {
		return c.DecodeError(r)
	}
	data, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading response: %s", err)
	}
	return errors.New(string(data))
}

func (c *Client) Parse(s string) error {
	fields, err := stringutil.SplitFields(s, ":")
	if err != nil {
		return err
	}
	switch len(fields) {
	case 1:
		c.Id = fields[0]
	case 2:
		c.Id = fields[0]
		c.Secret = fields[1]
	default:
		return fmt.Errorf("invalid number of fields: %d", len(fields))
	}
	return nil
}

// Clone returns a copy of the Client which uses the given context.
func (c *Client) Clone(ctx httpclient.Context) *Client {
	var cc Client
	if c != nil {
		cc = *c
	}
	cc.HTTPClient = cc.HTTPClient.Clone(ctx)
	return &cc
}

// Authorization returns the URL for requesting authorization from the user. Note that most
// providers require redirectURI to be registered with them.
func (c *Client) Authorization(redirectURI string, scopes []string, state string) string {
	data := make(url.Values)
	data.Set("client_id", c.Id)
	data.Set("redirect_uri", redirectURI)
	for k, v := range c.AuthorizationParameters {
		data.Set(k, v)
	}
	if len(scopes) > 0 {
		sep := ","
		if c.ScopeSeparator != "" {
			sep = c.ScopeSeparator
		}
		data.Set("scope", strings.Join(scopes, sep))
	}
	data.Set("state", state)
	return urlutil.AppendQuery(c.AuthorizationURL, data)
}

// Exchange exchanges the given code for a *Token. Note that redirectURI must
// match the value used in Authorization().
func (c *Client) Exchange(redirectURI string, code string) (*Token, error) {
	data := make(url.Values)
	data.Set("client_id", c.Id)
	data.Set("client_secret", c.Secret)
	data.Set("redirect_uri", redirectURI)
	data.Set("code", code)
	for k, v := range c.ExchangeParameters {
		data.Set(k, v)
	}
	resp, err := c.client().PostForm(c.ExchangeURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if c.responseHasError(resp) {
		return nil, c.decodeError(resp)
	}
	return NewToken(resp)
}

func (c *Client) do(f func(string, url.Values) (*httpclient.Response, error), u string, form url.Values, accessToken string) (*httpclient.Response, error) {
	data := make(url.Values)
	for k, v := range form {
		data[k] = v
	}
	if accessToken != "" {
		data.Set("access_token", accessToken)
	}
	resp, err := f(u, data)
	if err != nil {
		return nil, err
	}
	if c.responseHasError(resp) {
		defer resp.Close()
		return nil, c.decodeError(resp)
	}
	return resp, nil
}

// Get sends an oAuth 2 GET request.
func (c *Client) Get(u string, form url.Values, accessToken string) (*httpclient.Response, error) {
	return c.do(c.client().GetForm, u, form, accessToken)
}

// Get sends an oAuth 2 POST request.
func (c *Client) Post(u string, form url.Values, accessToken string) (*httpclient.Response, error) {
	return c.do(c.client().PostForm, u, form, accessToken)
}
