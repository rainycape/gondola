package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"gnd.la/util"
)

var (
	// Tried to encrypt a value without specifying a key.
	ErrNoEncryptionKey = errors.New("no encryption key specified")
	// Could not decrypt encrypted value (it was probably tampered with).
	ErrCouldNotDecrypt = errors.New("could not decrypt value")
)

// Cipherer is a function type which returns a cipher.Block
// from a given key.
type Cipherer func(key []byte) (cipher.Block, error)

// Encrypter performs symmetrical encryption and decryption
// using a block cipher in CTR mode and a random IV.
// It's important to note that the output from Encrypt
// must also be authenticated in other to be secure. One easy
// way to the that is using Signer.
type Encrypter struct {
	// Cipherer returns a cipher.Block initialized with
	// the given key. If nil, it defaults to aes.NewCipher
	Cipherer Cipherer
	// Key is the encryption key. If empty, all public methods
	// will return ErrNoEncryptionKey.
	Key []byte
}

func (e *Encrypter) getCipher() (cipher.Block, error) {
	if len(e.Key) == 0 {
		return nil, ErrNoEncryptionKey
	}
	cipherer := e.Cipherer
	if cipherer == nil {
		cipherer = aes.NewCipher
	}
	return cipherer(e.Key)
}

// Encrypt encrypts the given data using the Encrypter's
// Cipherer and Key, using a random initialization vector.
func (e *Encrypter) Encrypt(data []byte) ([]byte, error) {
	ci, err := e.getCipher()
	if err != nil {
		return nil, err
	}
	bs := ci.BlockSize()
	out := make([]byte, bs+len(data))
	iv := out[:bs]
	copy(iv, util.RandomBytes(bs))
	stream := cipher.NewCTR(ci, iv)
	stream.XORKeyStream(out[bs:], data)
	return out, nil
}

// Decrypt decrypts the given data using the Encrypter's
// Cipherer and Key.
func (e *Encrypter) Decrypt(data []byte) ([]byte, error) {
	ci, err := e.getCipher()
	if err != nil {
		return nil, err
	}
	bs := ci.BlockSize()
	if len(data) <= bs {
		return nil, ErrCouldNotDecrypt
	}
	iv, in := data[:bs], data[bs:]
	stream := cipher.NewCTR(ci, iv)
	out := make([]byte, len(in))
	stream.XORKeyStream(out, in)
	return out, nil
}
