// Package textutil contains small utility functions for parsing text.
package textutil

import (
	"bytes"
	"fmt"
	"unicode"
)

const (
	stateValue = iota
	stateValueQuoted
	stateEscape
)

// SplitFields separates the given text into multiple fields, using
// any character in sep as separator between fields. Additionally,
// fields using a separator character in their values might be
// quoted using any character in the quotes parameter. Any separator
// or quoting character might also be escaped by prepending a \ to it.
// Also, whitespaces between fields are ignored (if you want a field
// starting or ending with spaces, quote it).
func SplitFields(text string, sep string, quotes string) ([]string, error) {
	state := stateValue
	var curQuote rune
	var prevState int
	var values []string
	sepMap := make(map[rune]bool, len(sep))
	for _, v := range sep {
		sepMap[v] = true
	}
	quotesMap := make(map[rune]bool, len(quotes))
	for _, v := range quotes {
		quotesMap[v] = true
	}
	var buf bytes.Buffer
	for ii, v := range text {
		if state == stateEscape {
			if !sepMap[v] && !quotesMap[v] {
				return nil, fmt.Errorf("invalid escape sequence \\%s at %d", string(v), ii)
			}
			buf.WriteRune(v)
			state = prevState
			continue
		}
		switch {
		case v == '\\':
			prevState = state
			state = stateEscape
		case sepMap[v] && state != stateValueQuoted:
			values = append(values, buf.String())
			buf.Reset()
		case quotesMap[v]:
			if state == stateValueQuoted {
				if v == curQuote {
					state = stateValue
				} else {
					buf.WriteRune(v)
				}
			} else if buf.Len() == 0 {
				curQuote = v
				state = stateValueQuoted
			} else {
				buf.WriteRune(v)
			}
		default:
			if buf.Len() == 0 && state != stateValueQuoted && unicode.IsSpace(v) {
				continue
			}
			buf.WriteRune(v)
		}
	}
	if buf.Len() > 0 {
		if state == stateValueQuoted {
			return nil, fmt.Errorf("unfinished quoted value %q", buf.String())
		}
		values = append(values, buf.String())
	}
	return values, nil
}
