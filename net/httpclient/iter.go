package httpclient

import (
	"errors"
	"fmt"
	"net/http"

	"gnd.la/util/urlutil"
)

// Iter allows iterating over a chain of redirects until
// reaching a Response which is not a redirect.
type Iter struct {
	c    *Client
	req  *http.Request
	resp *Response
	err  error
	done bool
}

// Next returns true iff the Iter could perform a trip
// using Client.Trip with the current *http.Request.
// Calling Next() closes the previous *Response automatically
// if it was a redirect.
func (i *Iter) Next() bool {
	if i.err != nil {
		return false
	}
	if i.resp == nil {
		// First request, do some sanity checks
		if i.req.Method == "POST" || i.req.Method == "PUT" {
			i.err = fmt.Errorf("can't iter over Request with method %s", i.req.Method)
			return false
		}
		if i.req.Body != nil {
			i.err = errors.New("can't iter over Request with a body")
			return false
		}
	} else {
		// 2nd and subsequent requests
		if !i.resp.IsRedirect() {
			return false
		}
		i.resp.Close()
		location := i.resp.Header.Get("Location")
		next, err := urlutil.Join(i.req.URL.String(), location)
		if err != nil {
			i.err = err
			return false
		}
		req, err := http.NewRequest(i.req.Method, next, nil)
		if err != nil {
			i.err = err
			return false
		}
		i.req = req
	}
	resp, err := i.c.Trip(i.req)
	if err != nil {
		i.err = err
		return false
	}
	i.resp = resp
	return resp.IsRedirect()
}

// Request returns the current *http.Request, if any.
func (i *Iter) Request() *http.Request {
	return i.req
}

// Response returns the current *Response, if any.
func (i *Iter) Response() *Response {
	return i.resp
}

// Close closes the last *Response returned by
// the Iter.
func (i *Iter) Close() error {
	if i.resp != nil {
		return i.resp.Close()
	}
	return nil
}

// Err returns the latest error produced while executing
// an *http.Request by this Iter.
func (i *Iter) Err() error {
	return i.err
}

// Assert panics if i.Err() returns non-nil.
func (i *Iter) Assert() {
	if i.err != nil {
		panic(i.err)
	}
}
