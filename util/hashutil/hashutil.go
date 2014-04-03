// Package hashutil provides utility functions for hashing data.
//
// Functions in this package accept an interface{} argument, which
// must be one of the following:
//
//  - Any type implementing io.Reader
//  - string or *string
//  - []byte
//
// Anything else will panic at runtime.
package hashutil

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
	"io"
)

func _hash(h hash.Hash, src interface{}) string {
	switch x := src.(type) {
	case string:
		h.Write([]byte(x))
	case *string:
		h.Write([]byte(*x))
	case []byte:
		h.Write(x)
	case io.Reader:
		io.Copy(h, x)
	default:
		panic(fmt.Errorf("type %T can't be hashed by hashutil", src))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func _chash(h crypto.Hash, src interface{}) string {
	if !h.Available() {
		panic(fmt.Errorf("Hash %v is not available", h))
	}
	return _hash(h.New(), src)
}

// Md5 returns the MD5 hash as a string.
func Md5(src interface{}) string {
	return _chash(crypto.MD5, src)
}

// Sha1 returns the SHA1 hash as a string.
func Sha1(src interface{}) string {
	return _chash(crypto.SHA1, src)
}

// Sha224 returns the SHA224 hash as a string.
func Sha224(src interface{}) string {
	return _chash(crypto.SHA224, src)
}

// Sha256 returns the SHA256 hash as a string.
func Sha256(src interface{}) string {
	return _chash(crypto.SHA256, src)
}

// Sha512 returns the SHA512 hash as a string.
func Sha512(src interface{}) string {
	return _chash(crypto.SHA512, src)
}

// Adler32 returns the Adler-32 hash as a string.
func Adler32(src interface{}) string {
	return _hash(adler32.New(), src)
}

// CRC32 returns the CRC-32 hash as a string, using the IEEE polynomial.
func CRC32(src interface{}) string {
	return _hash(crc32.NewIEEE(), src)
}

func _crc64(poly uint64, src interface{}) string {
	return _hash(crc64.New(crc64.MakeTable(poly)), src)
}

// CRC64ISO returns the CRC-64 hash as a string, using the ISO polynomial.
func CRC64ISO(src interface{}) string {
	return _crc64(crc64.ISO, src)
}

// CRC64ECMA returns the CRC-64 hash as a string, using the ECMA polynomial.
func CRC64ECMA(src interface{}) string {
	return _crc64(crc64.ECMA, src)
}

// Fnv32 returns the fnv-1 32 bits hash as a string.
func Fnv32(src interface{}) string {
	return _hash(fnv.New32(), src)
}

// Fnv32a returns the fnv-1a 32 bits hash as a string.
func Fnv32a(src interface{}) string {
	return _hash(fnv.New32a(), src)
}

// Fnv64 returns the fnv-1 64 bits hash as a string.
func Fnv64(src interface{}) string {
	return _hash(fnv.New64(), src)
}

// Fnv64a returns the fnv-1a 64 bits hash as a string.
func Fnv64a(src interface{}) string {
	return _hash(fnv.New64a(), src)
}
