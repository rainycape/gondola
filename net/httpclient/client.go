package httpclient

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// DefaultTimeout is the default timeout for Transport instances.
	DefaultTimeout = 30 * time.Second
	// DefaultUserAgent is the default user agent for Transport instances.
	DefaultUserAgent = "Mozilla/5.0 (compatible; Gondola/1.0; +http://www.gondolaweb.com)"
)

// Context is the interface required to instantiate a *Client.
type Context interface {
	// Request returns the current in-flight request.
	Request() *http.Request
}

type Client struct {
	transport *transport
	c         *http.Client
}

// New returns a new *Client. The ctx parameter will usually be
// the *app.Context received by the handler which is calling this
// function. Note that passing a nil ctx will work under some circunstances
// but will fail when running on App Engine, so it's advisable to always
// pass an *app.Context to this function, for portability.
func New(ctx Context) *Client {
	tr := newTransport(ctx)
	client := &Client{
		transport: tr,
		c:         &http.Client{Transport: tr},
	}
	client.SetUserAgent(DefaultUserAgent)
	client.SetTimeout(DefaultTimeout)
	return client
}

func (c *Client) Clone(ctx Context) *Client {
	if c == nil || c.transport == nil {
		return New(ctx)
	}
	tr := c.transport.clone(ctx)
	cp := &Client{
		transport: tr,
		c:         &http.Client{Transport: tr},
	}
	if proxy := c.Proxy(); proxy != nil {
		cp.SetProxy(proxy)
	}
	return cp
}

// UserAgent returns the default user agent sent by requests
// without an User-Agent header set.
func (c *Client) UserAgent() string {
	return c.transport.UserAgent()
}

// SetUserAgent sets the default user agent.
func (c *Client) SetUserAgent(ua string) *Client {
	c.transport.SetUserAgent(ua)
	return c
}

// Timeout returns the timeout for this transport.
// The timeout represents the maximum total time for
// the request, including DNS resolution, connection
// establishment and the time spent reading the response
// body. Client initializes this value to DefaultTimeout.
func (c *Client) Timeout() time.Duration {
	return c.transport.Timeout()
}

// SetTimeout sets the timeout for requests sent by this client.
// Setting it to 0 disables timeouts.
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.transport.SetTimeout(timeout)
	return c
}

// HTTPClient returns the *http.Client used by this Client.
func (c *Client) HTTPClient() *http.Client {
	return c.c
}

// Transport returns the Transport used by this Client.
func (c *Client) Transport() Transport {
	return c.transport
}

// Get is a wrapper around http.Client.Get, returning a Response rather than an
// http.Response. See http.Client.Get for further details.
func (c *Client) Get(url string) (*Response, error) {
	return makeResponse(c.c.Get(url))
}

// Head is a wrapper around http.Client.Head, returning a Response rather than an
// http.Response. See http.Client.Head for further details.
func (c *Client) Head(url string) (*Response, error) {
	return makeResponse(c.c.Head(url))
}

// GetForm appends the given data to the given url and performs a GET
// request with it. See Get for further details.
func (c *Client) GetForm(url string, data url.Values) (*Response, error) {
	encoded := data.Encode()
	if encoded != "" {
		if strings.Contains(url, "?") {
			url += "&" + encoded
		} else {
			url += "?" + encoded
		}
	}
	return c.Get(url)
}

// Post is a wrapper around http.Client.Post, returning a Response rather than an
// http.Response. See http.Client.Post for further details.
func (c *Client) Post(url string, bodyType string, body io.Reader) (*Response, error) {
	return makeResponse(c.c.Post(url, bodyType, body))
}

// PostForm is a wrapper around http.Client.PostForm, returning a Response rather than an
// http.Response. See http.Client.PostForm for further details.
func (c *Client) PostForm(url string, data url.Values) (*Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Do is a wrapper around http.Client.Do, returning a Response rather than an
// http.Response. See http.Client.Do for further details.
func (c *Client) Do(req *http.Request) (*Response, error) {
	return makeResponse(c.c.Do(req))
}

// Trip performs a roundtrip with the given http.Request, without following
// any redirects, and returns the first response, which might be a redirect.
// It's basically a shorthand for c.Transport().RoundTrip(req).
func (c *Client) Trip(req *http.Request) (*Response, error) {
	return makeResponse(c.transport.RoundTrip(req))
}

// Proxy returns the Proxy function for this client, if any. Note that
// if SupportsProxy() returns false, this function will always return
// nil.
func (c *Client) Proxy() Proxy {
	if pr, ok := c.transport.Underlying().(proxyRoundTripper); ok {
		return pr.Proxy()
	}
	return nil
}

// SetProxy sets the Proxy function for this client. Setting it to nil
// disables any previously set proxy function. Note that if SupportsProxy()
// returns false, this function is a no-op.
func (c *Client) SetProxy(proxy Proxy) *Client {
	if pr, ok := c.transport.Underlying().(proxyRoundTripper); ok {
		pr.SetProxy(proxy)
	}
	return c
}

// SupportsProxy returns if the current runtime environement supports
// setting a proxy. Currently, this is false on App Engine and true
// otherwise.
func (c *Client) SupportsProxy() bool {
	_, ok := c.transport.Underlying().(proxyRoundTripper)
	return ok
}

// Iter returns an iterator for the given *http.Request. See Iter for
// further details
func (c *Client) Iter(req *http.Request) *Iter {
	return &Iter{c: c, req: req}
}

func makeResponse(r *http.Response, err error) (*Response, error) {
	if err != nil {
		return nil, err
	}
	return &Response{Response: r}, nil
}
