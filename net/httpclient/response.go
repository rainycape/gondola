package httpclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"gnd.la/net/urlutil"
)

// Response is a thin wrapper around *http.Response,
// adding a few conveniency methods.
type Response struct {
	*http.Response
	dec *json.Decoder
}

// IsOK returns true iff the response status code is
// >= 200 and < 300.
func (r *Response) IsOK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsRedirect returns true if the Response is a redirect
// (status codes 301, 302, 303 and 307).
func (r *Response) IsRedirect() bool {
	return r.StatusCode == http.StatusMovedPermanently ||
		r.StatusCode == http.StatusFound ||
		r.StatusCode == http.StatusSeeOther ||
		r.StatusCode == http.StatusTemporaryRedirect
}

// Redirect returns the redirect target as an absolute URL.
// If the Response is not a redirect of the pointed URL is
// not valid, an error is returned.
func (r *Response) Redirect() (string, error) {
	if r.IsRedirect() {
		location := r.Header.Get("Location")
		return urlutil.Join(r.Request.URL.String(), location)
	}
	return "", fmt.Errorf("response is not a redirect (status code %d)", r.StatusCode)
}

// ReadAll is a shorthand for ioutil.ReadAll(r.Body).
func (r *Response) ReadAll() ([]byte, error) {
	return ioutil.ReadAll(r.Body)
}

// URL is a shorthand for r.Request.URL.
func (r *Response) URL() *url.URL {
	return r.Request.URL
}

// Cookie returns the value for the first Cookie with the given
// name, or the empty string if no such cookie exists.
func (r *Response) Cookie(name string) string {
	for _, v := range r.Cookies() {
		if v.Name == name {
			return v.Value
		}
	}
	return ""
}

func (r *Response) String() string {
	return fmt.Sprintf("%T from %s (status code %d)", r, r.Request.URL, r.StatusCode)
}

// UnmarshalJSON reads the whole response body and decodes it as
// JSON in the provided out parameter using json.Unmarshal. If you
// need to sequentially decode several objects (e.g. a streaming
// response), see DecodeJSON.
func (r *Response) UnmarshalJSON(out interface{}) error {
	data, err := r.ReadAll()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

// DecodeJSON uses a json.Decoder to read and decode the next
// JSON-encoded value from the response body and stores it
// in the value pointed to by out.
func (r *Response) DecodeJSON(out interface{}) error {
	if r.dec == nil {
		r.dec = json.NewDecoder(r.Body)
	}
	return r.dec.Decode(out)
}

// Close is a shorthand for r.Body.Close()
func (r *Response) Close() error {
	if r != nil && r.Body != nil {
		return r.Body.Close()
	}
	return nil
}
