// Package unidecode implements a unicode transliterator
// which replaces non-ASCII characters with their ASCII
// approximations.
package unidecode

import (
	"unicode"
)

const pooledCapacity = 64

var pool = make(chan []rune, 8)

// Unidecode implements a unicode transliterator, which
// replaces non-ASCII characters with their ASCII
// counterparts.
// Given an unicode encoded string, returns
// another string with non-ASCII characters replaced
// with their closest ASCII counterparts.
// e.g. Unicode("áéíóú") => "aeiou"
func Unidecode(s string) string {
	if !decoded {
		decodeTransliterations()
	}
	l := len(s)
	var r []rune
	if l > pooledCapacity {
		r = make([]rune, 0, len(s))
	} else {
		select {
		case r = <-pool:
			r = r[:0]
		default:
			r = make([]rune, 0, pooledCapacity)
		}
	}
	for _, c := range s {
		if c <= unicode.MaxASCII {
			r = append(r, c)
			continue
		}
		if c > unicode.MaxRune {
			/* Ignore reserved chars */
			continue
		}
		if d := transliterations[c]; d != nil {
			r = append(r, d...)
		}
	}
	if l <= pooledCapacity {
		select {
		case pool <- r:
		default:
		}
	}
	return string(r)
}
