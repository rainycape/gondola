package httpclient

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// DefaultDeadline is the default deadline for Transport instances.
	DefaultDeadline = 60 * time.Second
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
	client.SetDeadline(DefaultDeadline)
	return client
}

func (c *Client) Clone(ctx Context) *Client {
	if c == nil || c.transport == nil {
		return New(ctx)
	}
	tr := c.transport.clone(ctx)
	return &Client{
		transport: tr,
		c:         &http.Client{Transport: tr},
	}
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

// Deadline returns the Deadline for this transport.
// The deadline represents the maximum total time for
// the request, including DNS resolution, connection
// establishment and the time spent reading the response
// body. Client initializes this value to DefaultDeadline.
// See SetDeadline for further details.
func (c *Client) Deadline() time.Duration {
	return c.transport.Deadline()
}

// SetDeadline sets the deadline for requests sent by this client.
// Setting it to 0 disables timeouts.
func (c *Client) SetDeadline(deadline time.Duration) *Client {
	c.transport.SetDeadline(deadline)
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
