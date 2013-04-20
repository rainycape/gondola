package util

import (
	"crypto"
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"hash/fnv"
)

func _hash(h hash.Hash, b []byte) string {
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func _chash(h crypto.Hash, b []byte) string {
	if !h.Available() {
		panic(fmt.Errorf("Hash %s is not available", h))
	}
	return _hash(h.New(), b)
}

// Md5 returns the MD5 hash as a string
func Md5(b []byte) string {
	return _chash(crypto.MD5, b)
}

// Sha1 returns the SHA1 hash as a string
func Sha1(b []byte) string {
	return _chash(crypto.SHA1, b)
}

// Sha224 returns the SHA224 hash as a string
func Sha224(b []byte) string {
	return _chash(crypto.SHA224, b)
}

// Sha256 returns the SHA256 hash as a string
func Sha256(b []byte) string {
	return _chash(crypto.SHA256, b)
}

// Sha512 returns the SHA512 hash as a string
func Sha512(b []byte) string {
	return _chash(crypto.SHA512, b)
}

// Adler32 returns the Adler-32 hash as a string
func Adler32(b []byte) string {
	return _hash(adler32.New(), b)
}

// CRC32 returns the CRC-32 hash using the IEEE polynomial as a string
func CRC32(b []byte) string {
	return _hash(crc32.NewIEEE(), b)
}

func _crc64(poly uint64, b []byte) string {
	return _hash(crc64.New(crc64.MakeTable(poly)), b)
}

// CRC64ISO returns the CRC-64 hash using the ISO polynomial as a string
func CRC64ISO(b []byte) string {
	return _crc64(crc64.ISO, b)
}

// CRC64ECMA returns the CRC-64 hash using the ECMA polynomial as a string
func CRC64ECMA(b []byte) string {
	return _crc64(crc64.ECMA, b)
}

// Fnv32 returns the fnv-1 32 bits hash as a string
func Fnv32(b []byte) string {
	return _hash(fnv.New32(), b)
}

// Fnv32a returns the fnv-1a 32 bits hash as a string
func Fnv32a(b []byte) string {
	return _hash(fnv.New32a(), b)
}

// Fnv64 returns the fnv-1 64 bits hash as a string
func Fnv64(b []byte) string {
	return _hash(fnv.New64(), b)
}

// Fnv64a returns the fnv-1a 64 bits hash as a string
func Fnv64a(b []byte) string {
	return _hash(fnv.New64a(), b)
}
