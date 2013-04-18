package cookies

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/gob"
	"errors"
	"fmt"
	"gondola/base64"
	"net/http"
	"strings"
	"time"
)

const (
	MaxSize = 4096
)

var (
	ErrNoSecret     = errors.New("No secret specified. Please, use mux.SetSecret().")
	ErrTampered     = errors.New("The cookie value has been altered by the client")
	ErrCookieTooBig = errors.New("This cookie is too big. Maximum size is 4096 bytes.")
	Permanent       = time.Unix(2147483647, 0).UTC()
	deleteExpires   = time.Unix(0, 0).UTC()
)

type Options struct {
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
}

type Cookies struct {
	r        *http.Request
	w        http.ResponseWriter
	secret   string
	defaults *Options
}

func New(r *http.Request, w http.ResponseWriter, secret string, defaults *Options) *Cookies {
	if defaults == nil {
		defaults = Defaults()
	}
	return &Cookies{r, w, secret, defaults}
}

// Defaults returns the default coookie options, which are:
// Path: "/"
// Expires: Permanent (cookie never expires)
// To change the defaults, see gondola/mux/Mux.SetDefaultCookieOptions()
func Defaults() *Options {
	return &Options{
		Path:    "/",
		Expires: Permanent,
	}
}

func (c *Cookies) encode(value interface{}) (string, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(value)
	if err != nil {
		return "", err
	}
	return base64.Encode(buf.Bytes()), nil
}

func (c *Cookies) decode(data string, arg interface{}) error {
	b, err := base64.Decode(data)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(arg)
}

func (c *Cookies) set(name, value string, o *Options) error {
	if len(value) > MaxSize {
		return ErrCookieTooBig
	}
	if o == nil {
		// c.defaults is guaranteed to not be nil
		o = c.defaults
	}
	// TODO: Calculate MaxAge depending
	// on expires and vice-versa
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     o.Path,
		Domain:   o.Domain,
		Expires:  o.Expires,
		MaxAge:   o.MaxAge,
		Secure:   o.Secure,
		HttpOnly: o.HttpOnly,
	}
	c.SetCookie(cookie)
	return nil
}

func (c *Cookies) sign(value string) (string, error) {
	if c.secret == "" {
		return "", ErrNoSecret
	}
	hm := hmac.New(sha1.New, []byte(c.secret))
	hm.Write([]byte(value))
	// Encoding the signature in base64 rather than
	// hex saves ~25% (hex has a 2x overhead while
	// base64 has a 4/3x)
	return base64.Encode(hm.Sum(nil)), nil
}

// GetCookie returns the raw *http.Coookie with
// the given name
func (c *Cookies) GetCookie(name string) (*http.Cookie, error) {
	return c.r.Cookie(name)
}

// SetCookie sets the given *http.Cookie
func (c *Cookies) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.w, cookie)
}

// Get uses the cookie value with the given name to
// populate the argument. If the types don't match
// (e.g. the cookie was set to a string and you try
// to get an int), an error will be returned.
func (c *Cookies) Get(name string, arg interface{}) error {
	cookie, err := c.GetCookie(name)
	if err != nil {
		return err
	}
	return c.decode(cookie.Value, arg)
}

// Set sets the cookie with the given name and encodes
// the given value using gob. If the cookie size is
// bigger than 4096 bytes, it returns ErrCookieTooBig.
// The options used for the cookie are the mux defaults. If
// you need to use different options, use SetOpts().
func (c *Cookies) Set(name string, value interface{}) error {
	return c.SetOpts(name, value, nil)
}

// SetOpts works like Set(), but accepts an Options parameter.
func (c *Cookies) SetOpts(name string, value interface{}, o *Options) error {
	encoded, err := c.encode(value)
	if err != nil {
		return err
	}
	return c.set(name, encoded, o)
}

// GetSecure works like Get, but for cookies set with SetSecure().
// See SetSecure() for the guarantees made about the
// cookie value.
func (c *Cookies) GetSecure(name string, arg interface{}) error {
	cookie, err := c.GetCookie(name)
	if err != nil {
		return err
	}
	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 3 || parts[1] != "sha1" {
		return ErrTampered
	}
	encoded, signature := parts[0], parts[2]
	s, err := c.sign(encoded)
	if err != nil {
		return err
	}
	if len(s) != len(signature) || subtle.ConstantTimeCompare([]byte(s), []byte(signature)) != 1 {
		return ErrTampered
	}
	return c.decode(encoded, arg)
}

// SetSecure sets a tamper-proof cookie, using the mux
// secret to sign its value with HMAC-SHA1. If you
// haven't set a secret (via mux.SetSecret()), this
// function will return ErrNoSecret.
// The options used for the cookie are the mux defaults. If
// you need to use different options, use SetOpts().
func (c *Cookies) SetSecure(name string, value interface{}) error {
	return c.SetSecureOpts(name, value, nil)
}

// SetSecureOpts works like SetSecure(), but accepts an Options parameter.
func (c *Cookies) SetSecureOpts(name string, value interface{}, o *Options) error {
	encoded, err := c.encode(value)
	if err != nil {
		return err
	}
	signature, err := c.sign(encoded)
	if err != nil {
		return err
	}
	// Only support sha1 for now, but store that in the cookie
	// to allow us to change the hashing algorighm in the future
	// if required (or even let the user choose it)
	return c.set(name, fmt.Sprintf("%s:sha1:%s", encoded, signature), o)
}

// Delete deletes with cookie with the given name
func (c *Cookies) Delete(name string) {
	cookie := &http.Cookie{
		Name:    name,
		Path:    "/",
		Expires: deleteExpires,
		MaxAge:  -1,
	}
	c.SetCookie(cookie)
}
