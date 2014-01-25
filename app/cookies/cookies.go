// Package cookies contains helper functions for setting and
// retrieving cookies, including signed and encrypted ones.
//
// Cookie values are encoded and decoded using encoding/gob, so
// you must register any non-basic type that you want to store
// in a cookie, using encoding/gob.Register.
//
// Signed cookies are signed using HMAC-SHA1. Encrypted cookies
// are encrypted with AES and then signed with HMAC-SHA1.
package cookies

import (
	"errors"
	"gnd.la/encoding/base64"
	"gnd.la/encoding/codec"
	"gnd.la/util/cryptoutil"
	"net/http"
	"time"
)

const (
	// Maximum cookie size. See section 6.3 at http://www.ietf.org/rfc/rfc2109.txt.
	MaxSize = 4096
)

var (
	// Cookie is too big. See MaxSize.
	ErrCookieTooBig = errors.New("cookie is too big (maximum size is 4096 bytes)")
	// Tried to use signed or encrypted cookies without a Signer.
	ErrNoSigner = errors.New("no signer specified")
	// Tried to use encrypted cookies without an Encrypter.
	ErrNoEncrypter = errors.New("no encrypter specified")

	// Maximum representable UNIX time with a signed 32 bit integer. This
	// means that cookies won't be really permanent, but they will expire
	// on January 19th 2038. I don't know about you, but I hope to be around
	// by that time, so hopefully I'll find a solution for this issue in the
	// next few years. See http://en.wikipedia.org/wiki/Year_2038_problem for
	// more information.
	Permanent      = time.Unix(2147483647, 0).UTC()
	deleteExpires  = time.Unix(0, 0).UTC()
	defaultCodec   = codec.Get("gob")
	cookieDefaults = &Options{Path: "/", Expires: Permanent}
)

// Options specify the default cookie Options used when setting
// a Cookie only by its name and value, like in Cookies.Set(),
// Cookies.SetSecure(), and Cookies.SetEncrypted().
//
// For more information about the cookie fields, see net/http.Cookie.
type Options struct {
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
}

// Cookies includes conveniency functions for setting
// and retrieving cookies. Use New() or gnd.la/app.Context.Cookies
// to create a Cookies instance.
type Cookies struct {
	r         *http.Request
	w         http.ResponseWriter
	c         *codec.Codec
	signer    *cryptoutil.Signer
	encrypter *cryptoutil.Encrypter
	defaults  *Options
}

type transformer func([]byte) ([]byte, error)

// New returns a new *Cookies object, which will read cookies from the
// given http.Request, and write them to the given http.ResponseWriter.
// Note that users will probably want to use gnd.la/app.Context.Cookies
// rather than this function to create a Cookies instance.
//
// The secret parameter is used for secure (signed) cookies, while
// encryptionKey is also used for encrypted cookies. If you don't use
// secure nor encrypted cookies, you might leave both parameters empty.
// If you only need signed cookies, you might leave encryptionKey
// empty.
//
// The default parameter specifies the default Options for the funcions
// which only take a name and a value. If you pass nil, Defaults will
// be used.
func New(r *http.Request, w http.ResponseWriter, c *codec.Codec, signer *cryptoutil.Signer, encrypter *cryptoutil.Encrypter, defaults *Options) *Cookies {
	if c == nil {
		c = defaultCodec
	}
	if defaults == nil {
		defaults = cookieDefaults
	}
	return &Cookies{r, w, c, signer, encrypter, defaults}
}

// Defaults returns the default coookie options, which are:
//
//  Path: "/"
//  Expires: Permanent (cookie never expires)
//
// To change the defaults, use SetDefaults.
func Defaults() *Options {
	return cookieDefaults
}

// SetDefaults changes the default cookie options.
func SetDefaults(defaults *Options) {
	if defaults == nil {
		defaults = &Options{}
	}
	cookieDefaults = defaults
}

func (c *Cookies) encode(value interface{}, t transformer) (string, error) {
	data, err := c.c.Encode(value)
	if err != nil {
		return "", err
	}
	if t != nil {
		data, err = t(data)
		if err != nil {
			return "", err
		}
	}
	return base64.Encode(data), nil
}

func (c *Cookies) decode(data string, arg interface{}, t transformer) error {
	b, err := base64.Decode(data)
	if err != nil {
		return err
	}
	if t != nil {
		b, err = t(b)
		if err != nil {
			return err
		}
	}
	return c.c.Decode(b, arg)
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

// setSigned sets a signed cookie from its data
func (c *Cookies) setSigned(name string, data []byte, o *Options) error {
	if c.signer == nil {
		return ErrNoSigner
	}
	signed, err := c.signer.Sign(data)
	if err != nil {
		return err
	}
	return c.set(name, signed, o)
}

// getSigned returns the signed cookie data if the signature is valid.
func (c *Cookies) getSigned(name string) ([]byte, error) {
	if c.signer == nil {
		return nil, ErrNoSigner
	}
	cookie, err := c.GetCookie(name)
	if err != nil {
		return nil, err
	}
	return c.signer.Unsign(cookie.Value)
}

// Has returns true if a cookie with the given name exists.
func (c *Cookies) Has(name string) bool {
	// TODO(hierro): This currently generates a *http.Cookie object
	// which is thrown away. Avoid that unnecessary allocation.
	cookie, _ := c.GetCookie(name)
	return cookie != nil
}

// GetCookie returns the raw *http.Coookie with
// the given name.
func (c *Cookies) GetCookie(name string) (*http.Cookie, error) {
	return c.r.Cookie(name)
}

// SetCookie sets the given *http.Cookie.
func (c *Cookies) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.w, cookie)
}

// Get uses the cookie value with the given name to
// populate the out argument. If the types don't match
// (e.g. the cookie was set to a string and you try
// to get an int), an error will be returned.
func (c *Cookies) Get(name string, out interface{}) error {
	cookie, err := c.GetCookie(name)
	if err != nil {
		return err
	}
	data, err := base64.Decode(cookie.Value)
	if err != nil {
		return err
	}
	return c.c.Decode(data, out)
}

// Set sets the cookie with the given name and encodes
// the given value using the codec provided in New. If
// the cookie size if bigger than 4096 bytes, it returns
// ErrCookieTooBig.
//
// The options used for the cookie are the default ones provided
// in New(), which will usually come from gnd.la/app.App.CookieOptions.
// If you need to specify different options, use SetOpts().
func (c *Cookies) Set(name string, value interface{}) error {
	return c.SetOpts(name, value, nil)
}

// SetOpts works like Set(), but accepts an Options parameter.
func (c *Cookies) SetOpts(name string, value interface{}, o *Options) error {
	data, err := c.c.Encode(value)
	if err != nil {
		return err
	}
	return c.set(name, base64.Encode(data), o)
}

// GetSecure works like Get, but for cookies set with SetSecure().
// See SetSecure() for the guarantees made about the
// cookie value.
func (c *Cookies) GetSecure(name string, out interface{}) error {
	data, err := c.getSigned(name)
	if err != nil {
		return err
	}
	return c.c.Decode(data, out)
}

// SetSecure sets a tamper-proof cookie, using the a
// *cryptoutil.Signer to sign its value. By default, it uses
// HMAC-SHA1. The user will be able to see
// the value of the cookie, but he will not be able
// to manipulate it. If you also require the value to be
// protected from being revealed to the user, use
// SetEncrypted().
//
// If you haven't set a Signer (usually set automatically for you, derived from
// the gnd.la/app.App.Secret field), this function will return an error.
//
// The options used for the cookie are the default ones provided
// in New(), which will usually come from gnd.la/app.App.CookieOptions.
// If you need to specify different options, use SetSecureOpts().
func (c *Cookies) SetSecure(name string, value interface{}) error {
	return c.SetSecureOpts(name, value, nil)
}

// SetSecureOpts works like SetSecure(), but accepts an Options parameter.
func (c *Cookies) SetSecureOpts(name string, value interface{}, o *Options) error {
	data, err := c.c.Encode(value)
	if err != nil {
		return err
	}
	return c.setSigned(name, data, o)
}

// GetEncrypted works like Get, but for cookies set with SetEncrypted().
// See SetEncrypted() for the guarantees made about the cookie value.
func (c *Cookies) GetEncrypted(name string, out interface{}) error {
	if c.encrypter == nil {
		return ErrNoEncrypter
	}
	data, err := c.getSigned(name)
	if err != nil {
		return err
	}
	decrypted, err := c.encrypter.Decrypt(data)
	if err != nil {
		return err
	}
	return c.c.Decode(decrypted, out)
}

// SetEncrypted sets a tamper-proof and encrypted cookie. The value is first
// encrypted using *cryptoutil.Encrypter and the signed using *cryptoutil.Signer.
// By default, these use AES and HMAC-SHA1 respectivelly. The user will not
// be able to tamper with the cookie value nor reveal its contents.
//
// If you haven't set a Signer (usually set automatically for you, derived from
// the gnd.la/app.App.Secret field) and an Encrypter (usually set automatically too,
// from gnd.la/app.App.EncryptionKey), this function will return an error.
//
// The options used for the cookie are the default ones provided
// in New(), which will usually come from gnd.la/app.App.CookieOptions.
// If you need to specify different options, use SetEncryptedOpts().
func (c *Cookies) SetEncrypted(name string, value interface{}) error {
	return c.SetEncryptedOpts(name, value, nil)
}

// SetEncryptedOpts works like SetEncrypted(), but accepts and Options parameter.
func (c *Cookies) SetEncryptedOpts(name string, value interface{}, o *Options) error {
	if c.encrypter == nil {
		return ErrNoEncrypter
	}
	data, err := c.c.Encode(value)
	if err != nil {
		return err
	}
	encrypted, err := c.encrypter.Encrypt(data)
	if err != nil {
		return err
	}
	return c.setSigned(name, encrypted, o)
}

// Delete deletes with cookie with the given name.
func (c *Cookies) Delete(name string) {
	cookie := &http.Cookie{
		Name:    name,
		Path:    "/",
		Expires: deleteExpires,
		MaxAge:  -1,
	}
	c.SetCookie(cookie)
}
