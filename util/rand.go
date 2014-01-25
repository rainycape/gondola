package util

import (
	"crypto/rand"
	"fmt"
	"io"
)

const (
	alphanumeric = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ0123456789"
	printable    = "!\"#$%'()*+,-./0123456789:;<=>?@ABCDEFGHJKLMNPQRSTUVWXYZ[\\]^_`abcdefghjkmnpqrstuvwxyz"
)

// Most code in this file is adapted from https://github.com/dchest/uniuri

// RandomString returns a random string with the given length.
// The alphabet used includes ASCII lowercase and uppercase
// letters and numbers.
func RandomString(length int) string {
	return randomString(length, alphanumeric)
}

// RandomPrintableString returns a random string with the
// given length, using all the ASCII printable characters
// as the alphabet.
func RandomPrintableString(length int) string {
	return randomString(length, printable)
}

func randomString(length int, chars string) string {
	b := make([]byte, length)
	r := make([]byte, length+(length/4)) // storage for random bytes.
	clen := byte(len(chars))
	maxrb := byte(256 - (256 % len(chars)))
	i := 0
	for {
		if _, err := io.ReadFull(rand.Reader, r); err != nil {
			panic(fmt.Errorf("error reading from random source: %s", err))
		}
		for _, c := range r {
			if c >= maxrb {
				// Skip this number to avoid modulo bias.
				continue
			}
			b[i] = chars[c%clen]
			i++
			if i == length {
				return string(b)
			}
		}
	}
	panic("unreachable")
}

// RandomBytes returns a slice of n random bytes.
func RandomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(fmt.Errorf("error reading from random source: %s", err))
	}
	return b
}
