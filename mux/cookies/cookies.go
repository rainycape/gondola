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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/gob"
	"errors"
	"fmt"
	"gnd.la/encoding/base64"
	"gnd.la/util"
	"net/http"
	"strings"
	"time"
)

const (
	MaxSize = 4096
)

var (
	ErrNoSecret        = errors.New("no secret specified")
	ErrNoEncryptionKey = errors.New("no encryption key specified")
	ErrCouldNotDecrypt = errors.New("could not decrypt value")
	ErrTampered        = errors.New("the cookie value has been altered by the client")
	ErrCookieTooBig    = errors.New("cookie is too big (maximum size is 4096 bytes)")
	Permanent          = time.Unix(2147483647, 0).UTC()
	deleteExpires      = time.Unix(0, 0).UTC()
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
// and retrieving cookies. Use New() or gnd.la/mux.Context.Cookies
// to create a Cookies instance.
type Cookies struct {
	r             *http.Request
	w             http.ResponseWriter
	secret        string
	encryptionKey string
	defaults      *Options
}

type transformer func([]byte) ([]byte, error)

// New returns a new *Cookies object, which will read cookies from the
// given http.Request, and write them to the given http.ResponseWriter.
// Note that users will probably want to use gnd.la/mux.Context.Cookies
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
func New(r *http.Request, w http.ResponseWriter, secret string, encryptionKey string, defaults *Options) *Cookies {
	if defaults == nil {
		defaults = Defaults()
	}
	return &Cookies{r, w, secret, encryptionKey, defaults}
}

// Defaults returns the default coookie options, which are:
//
//  Path: "/"
//  Expires: Permanent (cookie never expires)
//
// To change the defaults, see gnd.la/mux.Mux.SetDefaultCookieOptions()
func Defaults() *Options {
	return &Options{
		Path:    "/",
		Expires: Permanent,
	}
}

func (c *Cookies) encode(value interface{}, t transformer) (string, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(value)
	if err != nil {
		return "", err
	}
	encoded := buf.Bytes()
	if t != nil {
		encoded, err = t(encoded)
		if err != nil {
			return "", err
		}
	}
	return base64.Encode(encoded), nil
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

// sign signs the given value using HMAC-SHA1
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

// checkSignature splits the given value into the encoded
// and signature parts and verifies, in constant time, that
// the signature is correct.
func (c *Cookies) checkSignature(value string) (string, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 || parts[1] != "sha1" {
		return "", ErrTampered
	}
	encoded, signature := parts[0], parts[2]
	s, err := c.sign(encoded)
	if err != nil {
		return "", err
	}
	if len(s) != len(signature) || subtle.ConstantTimeCompare([]byte(s), []byte(signature)) != 1 {
		return "", ErrTampered
	}
	return encoded, nil
}

// setSigned sets a signed cookie, which should have been previously
// encoded
func (c *Cookies) setSigned(name string, value string, o *Options) error {
	signature, err := c.sign(value)
	if err != nil {
		return err
	}
	// Only support sha1 for now, but store that in the cookie
	// to allow us to change the hashing algorithm in the future
	// if required (or even let the user choose it)
	return c.set(name, fmt.Sprintf("%s:sha1:%s", value, signature), o)
}

func (c *Cookies) cipher() (cipher.Block, error) {
	if c.encryptionKey == "" {
		return nil, ErrNoEncryptionKey
	}
	return aes.NewCipher([]byte(c.encryptionKey))
}

func (c *Cookies) encrypter() (transformer, error) {
	ci, err := c.cipher()
	if err != nil {
		return nil, err
	}
	iv := []byte(util.RandomString(ci.BlockSize()))
	stream := cipher.NewCTR(ci, iv)
	return func(src []byte) ([]byte, error) {
		stream.XORKeyStream(src, src)
		return append(iv, src...), nil
	}, nil
}

func (c *Cookies) decrypter() (transformer, error) {
	ci, err := c.cipher()
	if err != nil {
		return nil, err
	}
	return func(src []byte) ([]byte, error) {
		bs := ci.BlockSize()
		if len(src) <= bs {
			return nil, ErrCouldNotDecrypt
		}
		iv, value := src[:bs], src[bs:]
		stream := cipher.NewCTR(ci, iv)
		stream.XORKeyStream(value, value)
		return value, nil
	}, nil
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
// populate the argument. If the types don't match
// (e.g. the cookie was set to a string and you try
// to get an int), an error will be returned.
func (c *Cookies) Get(name string, arg interface{}) error {
	cookie, err := c.GetCookie(name)
	if err != nil {
		return err
	}
	return c.decode(cookie.Value, arg, nil)
}

// Set sets the cookie with the given name and encodes
// the given value using gob. If the cookie size is
// bigger than 4096 bytes, it returns ErrCookieTooBig.
//
// The options used for the cookie are the default ones provided
// in New(), which will usually come from gnd.la/mux.Mux.DefaultCookieOptions.
// If you need to specify different options, use SetOpts().
func (c *Cookies) Set(name string, value interface{}) error {
	return c.SetOpts(name, value, nil)
}

// SetOpts works like Set(), but accepts an Options parameter.
func (c *Cookies) SetOpts(name string, value interface{}, o *Options) error {
	encoded, err := c.encode(value, nil)
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
	value, err := c.checkSignature(cookie.Value)
	if err != nil {
		return err
	}
	return c.decode(value, arg, nil)
}

// SetSecure sets a tamper-proof cookie, using the mux
// secret to sign its value with HMAC-SHA1. The user may
// find the value of the cookie, but he will not be able
// to manipulate it. If you also require the value to be
// protected from being revealed to the user, use
// SetEncrypted().
//
// If you haven't set a secret (usually via gnd.la/mux.Mux.SetSecret
// or using gnd.la/config), this function will return ErrNoSecret.
//
// The options used for the cookie are the default ones provided
// in New(), which will usually come from gnd.la/mux.Mux.DefaultCookieOptions.
// If you need to specify different options, use SetSecureOpts().
func (c *Cookies) SetSecure(name string, value interface{}) error {
	return c.SetSecureOpts(name, value, nil)
}

// SetSecureOpts works like SetSecure(), but accepts an Options parameter.
func (c *Cookies) SetSecureOpts(name string, value interface{}, o *Options) error {
	encoded, err := c.encode(value, nil)
	if err != nil {
		return err
	}
	return c.setSigned(name, encoded, o)
}

// GetEncrypted works like Get, but for cookies set with SetEncrypted().
// See SetEncrypted() for the guarantees made about the cookie value.
func (c *Cookies) GetEncrypted(name string, arg interface{}) error {
	cookie, err := c.GetCookie(name)
	if err != nil {
		return err
	}
	value, err := c.checkSignature(cookie.Value)
	if err != nil {
		return err
	}
	decrypter, err := c.decrypter()
	if err != nil {
		return err
	}
	return c.decode(value, arg, decrypter)
}

// SetEncrypted sets a tamper-proof and encrypted cookie. The value is first
// encrypted using AES and the signed using HMAC-SHA1. The user will not
// be able to tamper with the cookie value nor reveal its contents.
//
// If you haven't set a secret (usually via gnd.la/mux.Mux.SetSecret()
// or using gnd.la/config), this function will return ErrNoSecret.
//
// If you haven't set an encryption key (usually via gnd.la/mux.Mux.SetEncryptionKey()
// or gnd.la/config) this function will return ErrNoEncryptionKey.
//
// The options used for the cookie are the default ones provided
// in New(), which will usually come from gnd.la/mux.Mux.DefaultCookieOptions.
// If you need to specify different options, use SetEncryptedOpts().
func (c *Cookies) SetEncrypted(name string, value interface{}) error {
	return c.SetEncryptedOpts(name, value, nil)
}

// SetEncryptedOpts works like SetEncrypted(), but accepts and Options parameter.
func (c *Cookies) SetEncryptedOpts(name string, value interface{}, o *Options) error {
	encrypter, err := c.encrypter()
	if err != nil {
		return err
	}
	encoded, err := c.encode(value, encrypter)
	if err != nil {
		return err
	}
	return c.setSigned(name, encoded, o)
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
