package stringutil

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
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

// SplitError represents an error while splitting the fields. Note that not
// all errors returned for SplitFields are *SplitError (e.g. if the number of
// fields does not match ExactCount, the error is NOT an *SplitError).
type SplitError struct {
	// Pos indicates the position in the input while the error originated,
	// zero indexed.
	Pos int
	// Err is the original error.
	Err error
}

func (s *SplitError) Error() string {
	return fmt.Sprintf("index %d: %s", s.Pos, s.Err)
}

func newSplitError(text string, pos int, format string, args ...interface{}) *SplitError {
	e := fmt.Errorf(format, args...)
	return &SplitError{
		Pos: pos,
		Err: e,
	}
}

// SplitOptions represent options which can be specified when calling SplitFieldsOptions.
type SplitOptions struct {
	// Quotes includes the characters that are admitted as quote characters. If empty,
	// the default quoting characters ' and " are used. If you want
	// no quoting characters set this string to NO_QUOTES.
	Quotes string
	// ExactCount specifies the exact number of fields that
	// the text must have after splitting them. If the
	// number does not match, an error is returned.
	// Values <= 0 are ignored.
	ExactCount int
	// MaxSplits indicates the maximum number of splits performed. Id est,
	// MaxSplits = 1 will yield at most 2 fields. Values <= 0 are ignored.
	MaxSplits int
	// KeepQuotes indicates wheter to keep the quotes in the quoted fields.
	// Otherwise, quotes are removed from the fields.
	KeepQuotes bool
}

// SplitFieldsOptions works like SplitFields, but accepts an additional
// options parameter. See the type SplitOptions for the available options.
func SplitFieldsOptions(text string, sep string, opts *SplitOptions) ([]string, error) {
	quotes := "'\""
	if opts != nil {
		if opts.Quotes == NO_QUOTES {
			quotes = ""
		} else if opts.Quotes != "" {
			quotes = opts.Quotes
		}
	}
	state := stateValue
	var curQuote rune
	var quotePos int
	var prevState int
	var values []string
	isSep := makeSeparator(sep)
	isQuote := makeRuneChecker(quotes)
	var buf bytes.Buffer
	runes := []rune(text)
	for ii := 0; ii < len(runes); ii++ {
		v := runes[ii]
		if state == stateEscape {
			if !isSep(v) && !isQuote(v) && v != '\n' {
				quoted := strconv.Quote(string(v))
				return nil, newSplitError(text, ii, "invalid escape sequence \"\\%s\"", quoted[1:len(quoted)-1])
			}
			state = prevState
			buf.WriteRune(v)
			continue
		}
		switch {
		case v == '\\':
			prevState = state
			state = stateEscape
		case isSep(v) && state != stateValueQuoted:
			if buf.Len() > 0 || state == stateValueUnquoted {
				done := false
				if opts != nil && opts.MaxSplits > 0 && opts.MaxSplits == len(values) {
					buf.WriteString(string(runes[ii:]))
					done = true
				}
				s := buf.String()
				if state != stateValueUnquoted {
					s = strings.TrimRightFunc(s, unicode.IsSpace)
				}
				s = strings.TrimSuffix(s, NO_QUOTES)
				values = append(values, s)
				if done {
					return values, nil
				}
				buf.Reset()
				state = stateValue
			}
		case isQuote(v):
			if state == stateValueQuoted {
				if v == curQuote {
					if opts != nil && opts.KeepQuotes {
						buf.WriteRune(v)
					}
					state = stateValueUnquoted
					// write NO_QUOTES to the buffer, so we now
					// where to stop trimming
					buf.WriteString(NO_QUOTES)
				} else {
					buf.WriteRune(v)
				}
			} else if buf.Len() == 0 {
				curQuote = v
				quotePos = ii
				state = stateValueQuoted
				if opts != nil && opts.KeepQuotes {
					buf.WriteRune(v)
				}
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
			return nil, newSplitError(text, quotePos, "unclosed quote")
		}
		s := buf.String()
		if state != stateValueUnquoted {
			s = strings.TrimRightFunc(s, unicode.IsSpace)
		}
		s = strings.TrimSuffix(s, NO_QUOTES)
		if len(s) > 0 {
			values = append(values, s)
		}
	}
	if opts != nil && opts.ExactCount > 0 {
		if opts.ExactCount != len(values) {
			return nil, fmt.Errorf("invalid number of fields %d, must be %d", len(values), opts.ExactCount)
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

// SplitLines splits the given text into lines. Lines might be terminated
// by either "\r\n" (as in Windows) or just "\n" (as in Unix). Newlines might
// be escaped by prepending them with the '\' character.
func SplitLines(text string) []string {
	text = strings.Replace(text, "\r\n", "\n", -1)
	// replace escaped newlines
	text = strings.Replace(text, "\\\n", "", -1)
	return strings.Split(text, "\n")
}

// SplitCommonPrefix returns the common prefix in values and slice
// with each string in values with the common prefix removed.
func SplitCommonPrefix(values []string) (string, []string) {
	if len(values) == 0 {
		return "", values
	}
	minSize := -1
	for _, v := range values {
		if minSize < 0 || len(v) < minSize {
			minSize = len(v)
		}
	}
	split := 0
prefix:
	for ; split < minSize; split++ {
		ch := values[0][split]
		for _, v := range values[1:] {
			if v[split] != ch {
				break prefix
			}
		}
	}
	if split == 0 {
		return "", values
	}
	output := make([]string, len(values))
	for ii, v := range values {
		output[ii] = v[split:]
	}
	return values[0][:split], output
}

func makeRuneChecker(s string) func(rune) bool {
	m := make(map[rune]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return func(r rune) bool {
		_, ok := m[r]
		return ok
	}
}

func makeSeparator(sep string) func(rune) bool {
	if sep == "" {
		return unicode.IsSpace
	}
	return makeRuneChecker(sep)
}
