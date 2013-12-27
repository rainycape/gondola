package oauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Consumer struct {
	Key              string
	Secret           string
	Service          string
	RequestTokenURL  string
	AccessTokenURL   string
	AuthorizationURL string
	CallbackURL      string
}

func (c *Consumer) defaultParameters() url.Values {
	values := make(url.Values)
	values.Add("oauth_version", "1.0")
	values.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	values.Add("oauth_consumer_key", c.Key)
	values.Add("oauth_nonce", strconv.FormatInt(rand.Int63(), 10))
	values.Add("oauth_signature_method", "HMAC-SHA1")
	return values
}

func (c *Consumer) sign(method string, url string, values url.Values, secret string) string {
	base := fmt.Sprintf("%s&%s&%s", method, encode(url), encodePlusEncoded(values.Encode()))
	key := encode(c.Secret) + "&" + encode(secret)
	return c.digest(key, base)
}

// digest generates a HMAC-SHA1 signature with the given key and data.
func (c *Consumer) digest(key string, data string) string {
	h := hmac.New(sha1.New, []byte(key))
	// TODO: Check for errors here?
	io.WriteString(h, data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (c *Consumer) headers(method string, url string, values url.Values, secret string) map[string]string {
	signature := c.sign(method, url, values, secret)
	var headers []string
	for k, v := range values {
		if strings.HasPrefix(k, "oauth") {
			headers = append(headers, encodeQuoted(k, v[0]))
			values.Del(k)
		}
	}
	headers = append(headers, encodeQuoted("oauth_signature", signature))
	sort.Strings(headers)
	return map[string]string{
		"Authorization": "OAuth " + strings.Join(headers, ", "),
	}
}

// Authorization requests a Request Token and returns the URL the user should
// visit to authorize it as well as the token, which needs to be used later
// for exchanging it for an Access Token.
func (c *Consumer) Authorization() (string, *Token, error) {
	values := c.defaultParameters()
	values.Add("oauth_callback", c.CallbackURL)
	headers := c.headers("POST", c.RequestTokenURL, values, "")
	fmt.Println(headers, c.RequestTokenURL)
	resp, err := sendReq("POST", c.RequestTokenURL, headers, values)
	if err != nil {
		return "", nil, err
	}
	token, err := parseToken(resp)
	if err != nil {
		return "", nil, err
	}
	return c.AuthorizationURL + "?oauth_token=" + token.Key, token, nil
}

// Exchange exchanges a Request Token for an Access Token using the given
// verifier. The verifier is sent by the provider to the consumer at the
// callback URL. If the provider you're using doesn't require a verifier, just
// pass an empty string.
func (c *Consumer) Exchange(token *Token, verifier string) (*Token, error) {
	p := c.defaultParameters()
	p.Add("oauth_token", token.Key)
	if verifier != "" {
		p.Add("oauth_verifier", verifier)
	}
	headers := c.headers("POST", c.AccessTokenURL, p, token.Secret)
	resp, err := sendReq("POST", c.AccessTokenURL, headers, nil)
	if err != nil {
		return nil, err
	}
	return parseToken(resp)
}

// Get performs a GET request to the given URL with the given values and
// signed with the consumer and the given token (if any). The url parameter
// can't contain a query string. All url parameters should be passed using
// the values parameter.
func (c *Consumer) Get(url string, values url.Values, token *Token) (*http.Response, error) {
	return c.SendRequest("GET", url, values, token)
}

// Post performs a POST request to the given URL with the given values and
// signed with the given token (if any).
func (c *Consumer) Post(url string, values url.Values, token *Token) (*http.Response, error) {
	return c.SendRequest("POST", url, values, token)
}

// Request returns a *http.Request with the given method, url and values, which is already
// signed using the given token (if any).
func (c *Consumer) Request(method string, url string, values url.Values, token *Token) (*http.Request, error) {
	vals := c.defaultParameters()
	for k, v := range values {
		vals[k] = append(vals[k], v...)
	}
	vals.Add("oauth_token", token.Key)
	var secret string
	if token != nil {
		secret = token.Secret
	}
	headers := c.headers(method, url, vals, secret)
	return req(method, url, headers, vals)
}

// SendRequest works like Requests, but it also sends the request and
// returns an *http.Response.
func (c *Consumer) SendRequest(method string, url string, values url.Values, token *Token) (*http.Response, error) {
	r, err := c.Request(method, url, values, token)
	if err != nil {
		return nil, err
	}
	return client.Do(r)
}
