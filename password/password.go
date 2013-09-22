package password

import (
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"gnd.la/util"
	"strings"
)

var (
	ErrNoMatch             = errors.New("password does not match")
	ErrInvalidFieldCount   = errors.New("encoded password does not have 3 fields")
	ErrInvalidSaltLength   = errors.New("salt does not have the same length as the hash output")
	ErrInvalidHashedLength = errors.New("hashed password does not have the same length as the hash output")
	ErrInvalidHex          = errors.New("hashed password is not properly encoded")
)

// Password represents an encoded password, which can be stored
// as a string and then used to verify if the user provided
// password matches the stored one.
type Password string

func (p Password) field(idx int) string {
	s := -1
	c := 0
	for ; idx >= 0; idx-- {
		s += c + 1
		c = strings.Index(string(p)[s:], ":")
		if c < 0 {
			c = len(p) - s
			break
		}
	}
	return string(p)[s : s+c]
}

func (p Password) validate() ([]byte, Hash, error) {
	if strings.Count(string(p), ":") != 2 {
		return nil, 0, ErrInvalidFieldCount
	}
	h, err := p.Hash()
	if err != nil {
		return nil, 0, err
	}
	if len(p.Salt()) != h.Size() {
		return nil, 0, ErrInvalidSaltLength
	}
	decoded, err := hex.DecodeString(p.field(2))
	if err != nil {
		return nil, 0, ErrInvalidHex
	}
	if len(decoded) != h.Size() {
		return nil, 0, ErrInvalidHashedLength
	}
	return decoded, h, nil
}

// Salt returns the salt used to encode the password.
func (p Password) Salt() string {
	return p.field(1)
}

// Hash returns the Hash used to encode the password.
func (p Password) Hash() (Hash, error) {
	return HashNamed(p.field(0))
}

// String returns the password string as hash:salt:hex(hash)
func (p Password) String() string {
	return string(p)
}

// IsValid returns true iff the password is a correctly encoded password.
// This means it has a hash that is available and the salt and hashed
// data have the same length as the hash output.
func (p Password) IsValid() bool {
	_, _, err := p.validate()
	return err == nil
}

// Check returns nil if the password could be verified without
// any errors and it matches the provided plain text password.
// This function performs a constant time comparison,
// so it's not vulnerable to timing attacks.
func (p Password) Check(plain string) error {
	decoded, hash, err := p.validate()
	if err != nil {
		// This does not affect the time-constness of the function
		// since an invalid Password string will always return at
		// this point, regardless of the input.
		return err
	}
	h := hash.New()
	h.Write([]byte(p.Salt() + plain))
	if subtle.ConstantTimeCompare(decoded, h.Sum(nil)) != 1 {
		return ErrNoMatch
	}
	return nil
}

// Matches is a shorthand for Check(plain) == nil. Id est,
// if returns true iff the password is correctly encoded
// and the provided plain password matches the encoded
// one.
func (p Password) Matches(plain string) bool {
	return p.Check(plain) == nil
}

// New returns a new Password hashed using the default hash
// (at this time, sha256).
func New(plain string) Password {
	return NewHashed(plain, DEFAULT_HASH)
}

// NewHashed returns a password hashed with the given hash. If
// the hash is not available or not valid, it will panic.
func NewHashed(plain string, hash Hash) Password {
	// Use the same number of bits for the salt and the hash, since
	// it provides the maximum possible security.
	salt := util.RandomString(hash.Size())
	h := hash.New()
	h.Write([]byte(salt + plain))
	return Password(fmt.Sprintf("%s:%s:%s", hash.Name(), salt, hex.EncodeToString(h.Sum(nil))))
}
