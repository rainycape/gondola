package util

import (
	"crypto"
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/hex"
	"fmt"
)

func _hash(h crypto.Hash, b []byte) string {
	if !h.Available() {
		panic(fmt.Errorf("Hash %s is not available", h))
	}
	m := h.New()
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}

// Md5 returns the MD5 hash as a string
func Md5(b []byte) string {
	return _hash(crypto.MD5, b)
}

// Sha1 returns the SHA1 hash as a string
func Sha1(b []byte) string {
	return _hash(crypto.SHA1, b)
}

// Sha224 returns the SHA224 hash as a string
func Sha224(b []byte) string {
	return _hash(crypto.SHA224, b)
}

// Sha256 returns the SHA256 hash as a string
func Sha256(b []byte) string {
	return _hash(crypto.SHA256, b)
}

// Sha512 returns the SHA512 hash as a string
func Sha512(b []byte) string {
	return _hash(crypto.SHA512, b)
}
