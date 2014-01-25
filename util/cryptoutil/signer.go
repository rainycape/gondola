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
	Hasher Hasher
	Key    []byte
	Salt   []byte
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

func (s *Signer) Sign(data []byte) (string, error) {
	signature, err := s.sign(data)
	if err != nil {
		return "", err
	}
	return base64.Encode(data) + ":" + base64.Encode(signature), nil
}

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
