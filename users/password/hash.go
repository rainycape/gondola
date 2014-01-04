package password

import (
	"code.google.com/p/go.crypto/pbkdf2"
	"crypto"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
)

// Type Hash represents a hash algorithm for hashing passwords.
type Hash uint

const (
	SHA1         = Hash(crypto.SHA1)
	SHA224       = Hash(crypto.SHA224)
	SHA256       = Hash(crypto.SHA256)
	SHA384       = Hash(crypto.SHA384)
	SHA512       = Hash(crypto.SHA512)
	DEFAULT_HASH = SHA256
)

// Name returns the name of the hash.
func (h Hash) Name() string {
	switch h {
	case SHA1:
		return "sha1"
	case SHA224:
		return "sha224"
	case SHA256:
		return "sha256"
	case SHA384:
		return "sha384"
	case SHA512:
		return "sha512"
	}
	panic("invalid hash")
}

// Available returns if the given hash function is available at runtime.
func (h Hash) Available() bool {
	return crypto.Hash(h).Available()
}

// Size returns the size in bytes of the hash output.
func (h Hash) Size() int {
	return crypto.Hash(h).Size()
}

func (h Hash) RawHash(salt string, plain string, rounds int) []byte {
	return pbkdf2.Key([]byte(plain), []byte(salt), rounds, h.Size(), h.New)
}

func (h Hash) Hash(salt string, plain string, rounds int) string {
	raw := h.RawHash(salt, plain, rounds)
	return hex.EncodeToString(raw)
}

// New returns a new hash.Hash which calculates the given hash function.
func (h Hash) New() hash.Hash {
	return crypto.Hash(h).New()
}

// HashNamed returns the hash with the given name. If no hash
// with that name is found, an error is returned.
func HashNamed(name string) (Hash, error) {
	switch name {
	case "sha1":
		return SHA1, nil
	case "sha224":
		return SHA224, nil
	case "sha256":
		return SHA256, nil
	case "sha384":
		return SHA384, nil
	case "sha512":
		return SHA512, nil
	}
	return Hash(0), fmt.Errorf("no hash named %q", name)
}
