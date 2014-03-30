package cryptoutil

// EncryptSigner is a conveniency type
// which performs encryption with and then
// signs the encrypted data.
type EncryptSigner struct {
	// Encrypter used for encryption/decryption.
	Encrypter *Encrypter
	// Signer used for signing/unsigning.
	Signer *Signer
}

// EncryptSign encrypts the data using the EncryptSigner Encrypter
// and the signs it using its Signer.
func (e *EncryptSigner) EncryptSign(data []byte) (string, error) {
	enc, err := e.Encrypter.Encrypt(data)
	if err != nil {
		return "", err
	}
	return e.Signer.Sign(enc)
}

// UnsignDecrypt takes an encrypted and signed string, previously returned
// from EncryptSign, checks its signature and returns the decrypted data.
// If the signature does not match or the data can't be correctly decrypted
// an error is returned.
func (e *EncryptSigner) UnsignDecrypt(data string) ([]byte, error) {
	enc, err := e.Signer.Unsign(data)
	if err != nil {
		return nil, err
	}
	return e.Encrypter.Decrypt(enc)
}
