package oauth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

var (
	NoRedirectAllowedError = errors.New("redirects not allowed with oAuth")
	client                 = &http.Client{CheckRedirect: preventRedirect}
)

func preventRedirect(req *http.Request, via []*http.Request) error {
	return NoRedirectAllowedError
}

func req(method, url string, headers map[string]string, values url.Values) (*http.Request, error) {
	var req *http.Request
	var err error
	// We have to do all this dancing because due to the way
	// interfaces are implemented, if we define a variable
	// for the request body and not set it, NewRequest will
	// get an interface to a nil pointer rather than a "bare nil"
	// and will not work properly.
	if len(values) > 0 {
		if method == "GET" || method == "HEAD" {
			url = url + "?" + values.Encode()
			req, err = http.NewRequest(method, url, nil)
		} else {
			req, err = http.NewRequest(method, url, strings.NewReader(values.Encode()))
			if req != nil {
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			}
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	return req, nil
}

func sendReq(method, url string, headers map[string]string, values url.Values) (*http.Response, error) {
	r, err := req(method, url, headers, values)
	if err != nil {
		return nil, err
	}
	return client.Do(r)
}
