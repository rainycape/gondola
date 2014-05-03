package password

import (
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"gnd.la/util/stringutil"
	"strconv"
	"strings"
)

var (
	// ErrNoMatch means the password provided in Check() does not match the stored one.
	ErrNoMatch = errors.New("password does not match")
	// ErrInvalidFieldCount means the password does not have the required
	// number of fields.
	ErrInvalidFieldCount = errors.New("encoded password does not have 4 fields")
	// ErrInvalidRoundCount means the number of rounds stored in the password
	// is not a positive integer.
	ErrInvalidRoundCount = errors.New("invalid number of rounds")
	// ErrInvalidSaltLength means the salt stored with the password does
	// not match the password's hash key size.
	ErrInvalidSaltLength = errors.New("salt does not have the same length as the hash output")
	// ErrInvalidHashedLength the hash output stored in the password does
	// no match the password's hash output size.
	ErrInvalidHashedLength = errors.New("hashed password does not have the same length as the hash output")
	// ErrInvalidHex means the encoded password value is not properly
	// encoded as hexadecimal.
	ErrInvalidHex = errors.New("hashed password is not properly encoded")
	// Rounds is the number of PBKDF2 rounds used when creating a new password.
	// Altering this number won't break already generated and stored passwords,
	// since they store the number of rounds they were created with.
	Rounds = 4096
)

const (
	// MaxPasswordLength is the maximum password length. Trying to create
	// or verify a password longer than this will cause an error. This
	// is a measure against DoS attacks.
	MaxPasswordLength = 8192
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

func (p Password) validate() ([]byte, Hash, string, int, error) {
	if strings.Count(string(p), ":") != 3 {
		return nil, 0, "", 0, ErrInvalidFieldCount
	}
	h, err := p.Hash()
	if err != nil {
		return nil, 0, "", 0, err
	}
	rounds, err := p.Rounds()
	if err != nil {
		return nil, 0, "", 0, ErrInvalidRoundCount
	}
	salt := p.Salt()
	if len(salt) != h.Size() {
		return nil, 0, "", 0, ErrInvalidSaltLength
	}
	decoded, err := hex.DecodeString(p.field(3))
	if err != nil {
		return nil, 0, "", 0, ErrInvalidHex
	}
	if len(decoded) != h.Size() {
		return nil, 0, "", 0, ErrInvalidHashedLength
	}
	return decoded, h, salt, rounds, nil
}

// Salt returns the salt used to encode the password.
func (p Password) Salt() string {
	return p.field(2)
}

// Hash returns the Hash used to encode the password.
func (p Password) Hash() (Hash, error) {
	return HashNamed(p.field(0))
}

// Rounds returns the number of PBKDF2 rounds used to encode this password.
func (p Password) Rounds() (int, error) {
	r := p.field(1)
	return strconv.Atoi(r)
}

// String returns the password string as hash:rounds:salt:hex(hash)
func (p Password) String() string {
	return string(p)
}

// IsValid returns true iff the password is a correctly encoded password.
// This means it has a hash that is available and the salt and hashed
// data have the same length as the hash output.
func (p Password) IsValid() bool {
	_, _, _, _, err := p.validate()
	return err == nil
}

// Check returns nil if the password could be verified without
// any errors and it matches the provided plain text password.
// This function performs a constant time comparison,
// so it's not vulnerable to timing attacks.
func (p Password) Check(plain string) error {
	if len(plain) > MaxPasswordLength {
		return ErrNoMatch
	}
	decoded, hash, salt, rounds, err := p.validate()
	if err != nil {
		// This does not affect the time-constness of the function
		// since an invalid Password string will always return at
		// this point, regardless of the input.
		return err
	}
	if subtle.ConstantTimeCompare(decoded, hash.RawHash(salt, plain, rounds)) != 1 {
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
	if len(plain) > MaxPasswordLength {
		return Password("")
	}
	// Use the same number of bits for the salt and the hash, since
	// it provides the maximum possible security.
	salt := stringutil.Random(hash.Size())
	return Password(fmt.Sprintf("%s:%d:%s:%s", hash.Name(), Rounds, salt, hash.Hash(salt, plain, Rounds)))
}
