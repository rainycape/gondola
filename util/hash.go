package util

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
)

func Sha1(b []byte) string {
	m := sha1.New()
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}

func Md5(b []byte) string {
	m := md5.New()
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}
