package stringutil

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/dchest/uniuri"
)

var (
	alphanumeric = []byte("abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ0123456789")
	printable    = []byte("!\"#$%'()*+,-./0123456789:;<=>?@ABCDEFGHJKLMNPQRSTUVWXYZ[\\]^_`abcdefghjkmnpqrstuvwxyz")
)

// Random returns a random string with the given length.
// The alphabet used includes ASCII lowercase and uppercase
// letters and numbers.
func Random(length int) string {
	return randomString(length, alphanumeric)
}

// RandomPrintable returns a random string with the
// given length, using all the ASCII printable characters
// as the alphabet.
func RandomPrintable(length int) string {
	return randomString(length, printable)
}

func randomString(length int, chars []byte) string {
	return uniuri.NewLenChars(length, chars)
}

// RandomBytes returns a slice of n random bytes.
func RandomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(fmt.Errorf("error reading from random source: %s", err))
	}
	return b
}
