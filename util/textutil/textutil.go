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
	stateValueUnquoted
	stateEscape
)

const (
	// Unicode non-character. Used to signal that there are no
	// quoting characters.
	NO_QUOTES = "\uffff"
)

type SplitOptions struct {
	// Characters that are admitted as quote characters. If empty,
	// the default quoting characters ' and " are used. If you want
	// no quoting characters set this string to NO_QUOTES.
	Quotes string
	// If > 0 specifies the exact number of fields that
	// the text must have after splitting them. If the
	// number does not match, an error is returned.
	Count int
}

// SplitFieldsOptions works like SplitFields, but accepts an additional
// options parameter. See the type SplitOptions for the available options.
func SplitFieldsOptions(text string, sep string, opts *SplitOptions) ([]string, error) {
	quotes := "'\""
	if opts != nil {
		if opts.Quotes != NO_QUOTES {
			quotes = opts.Quotes
		} else {
			quotes = ""
		}
	}
	state := stateValue
	var curQuote rune
	var prevState int
	var values []string
	isSep := makeSeparator(sep)
	quotesMap := make(map[rune]bool, len(quotes))
	for _, v := range quotes {
		quotesMap[v] = true
	}
	var buf bytes.Buffer
	for ii, v := range text {
		if state == stateEscape {
			if isSep(v) && !quotesMap[v] {
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
		case isSep(v) && state != stateValueQuoted:
			if buf.Len() > 0 || state == stateValueUnquoted {
				values = append(values, buf.String())
				buf.Reset()
				state = stateValue
			}
		case quotesMap[v]:
			if state == stateValueQuoted {
				if v == curQuote {
					state = stateValueUnquoted
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
			if buf.Len() == 0 && state != stateValueQuoted && (isSep(v) || unicode.IsSpace(v)) {
				continue
			}
			buf.WriteRune(v)
			if state == stateValueUnquoted {
				state = stateValue
			}
		}
	}
	if buf.Len() > 0 || state == stateValueUnquoted {
		if state == stateValueQuoted {
			return nil, fmt.Errorf("unfinished quoted value %q", buf.String())
		}
		values = append(values, buf.String())
	}
	if opts != nil && opts.Count > 0 {
		if opts.Count != len(values) {
			return nil, fmt.Errorf("invalid number of fields %d, must be %d", len(values), opts.Count)
		}
	}
	return values, nil
}

// SplitFields separates the given text into multiple fields, using
// any character in sep as separator between fields. Additionally,
// fields using a separator character in their values might be
// quoted using ' or " (this can be changed with SplitFieldsOptions).
// Any separator or quoting character might also be escaped by prepending
// a \ to it. Also, whitespaces between fields are ignored (if you want
// a field starting or ending with spaces, quote it).
func SplitFields(text string, sep string) ([]string, error) {
	return SplitFieldsOptions(text, sep, nil)
}

func makeSeparator(sep string) func(rune) bool {
	if sep == "" {
		return unicode.IsSpace
	}
	sepMap := make(map[rune]struct{}, len(sep))
	for _, v := range sep {
		sepMap[v] = struct{}{}
	}
	return func(r rune) bool {
		_, ok := sepMap[r]
		return ok
	}
}
