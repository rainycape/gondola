package cryptoutil

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"errors"
	"gnd.la/encoding/base64"
	"hash"
	"strings"
)

var (
	// Tried to sign with an empty key.
	ErrNoSigningKey = errors.New("no signing key specified")
	// The value does not seem to be signed.
	ErrNotSigned = errors.New("value does look like it was signed")
	// Signature does not match (definitely tampered).
	ErrTampered = errors.New("the value has been tampered with")
)

// Hasher represents a function type which returns a hash.Hash
// with the given key.
type Hasher func(key []byte) (hash.Hash, error)

// Signer signs messages using the provided Hasher function,
// Key and, optionally but highly recommended, a Salt. The
// data and the signature are base64 encoded, removing any
// padding ('=') characters, so its output can be safely
// used in cookies, urls or headers.
// Note that you MUST use a unique salt in every system of your
// app which uses a Signer, otherwise you might be exposing yourself
// to security risks (e.g. if you use signing in your forms and in some
// authentication system, it's ok to use the same salt for all the forms, but
// not to use the same salt for the forms and the authentication).
type Signer struct {
	// Hasher is the function used to obtain a hash.Hash from the
	// Key.
	Hasher Hasher
	// Key is the key used for signing the data.
	Key []byte
	// Salt is prepended to the value to be signed. See the Signer
	// documentation for security considerations about the salt.
	Salt []byte
}

func (s *Signer) sign(data []byte) ([]byte, error) {
	if len(s.Key) == 0 {
		return nil, ErrNoSigningKey
	}
	var h hash.Hash
	var err error
	if s.Hasher != nil {
		h, err = s.Hasher(s.Key)
		if err != nil {
			return nil, err
		}
	} else {
		h = hmac.New(sha1.New, s.Key)
	}
	if len(s.Salt) > 0 {
		if _, err := h.Write(s.Salt); err != nil {
			return nil, err
		}
	}
	if _, err := h.Write(data); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// Sign signs the given data and returns the data plus the signature
// as a string. See Signer documentation for the characteristics of
// the returned string.
func (s *Signer) Sign(data []byte) (string, error) {
	signature, err := s.sign(data)
	if err != nil {
		return "", err
	}
	return base64.Encode(data) + ":" + base64.Encode(signature), nil
}

// Unsign takes a string, previously returned from Sign, checks
// that its signature is valid and, in that case, returns the initial
// data. If the signature is not valid, an error is returned.
func (s *Signer) Unsign(signed string) ([]byte, error) {
	parts := strings.Split(signed, ":")
	if len(parts) != 2 {
		return nil, ErrNotSigned
	}
	data, err := base64.Decode(parts[0])
	if err != nil {
		return nil, err
	}
	signature, err := base64.Decode(parts[1])
	if err != nil {
		return nil, err
	}
	sign, err := s.sign(data)
	if err != nil {
		return nil, err
	}
	if len(sign) != len(signature) || subtle.ConstantTimeCompare(sign, signature) != 1 {
		return nil, ErrTampered
	}
	return data, nil
}
