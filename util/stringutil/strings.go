package stringutil

import (
	"io/ioutil"
	"strings"
	"unicode"
)

// StringFunc is a function which accepts an string and returns another string.
type StringFunc func(string) string

// UnCamelCase separates the a camel-cased string into lowercase
// using sep as the separator between words. Multiple uppercase
// characters together are treated as a single word (e.g. TESTFoo
// is interpreted as the words 'TEST' and 'Foo', while FooBAR is
// intepreted as 'Foo' and 'BAR').
func UnCamelCase(s string) []string {
	ls := len(s)
	switch ls {
	case 0:
		return nil
	case 1:
		return []string{s}
	}
	runes := []rune(s)
	var words []string
	idx := 0
	for ii, v := range runes[1 : ls-1] {
		if unicode.IsUpper(v) && (unicode.IsLower(runes[ii+2]) || unicode.IsLower(runes[ii])) {
			n := ii + 1
			words = append(words, string(runes[idx:n]))
			idx = n
		}
	}
	// Append last word
	words = append(words, string(runes[idx:]))
	return words
}

// CamelCaseToString transform a camel-cased string into a
// string containing the words in the original string, separated
// by sep. Optionally, a f argument might be provided, which will
// be used to transform the original strings before concatenating
// them into the final string.
func CamelCaseToString(s string, sep string, f StringFunc) string {
	words := UnCamelCase(s)
	if f != nil {
		w := make([]string, len(words))
		for ii, v := range words {
			w[ii] = f(v)
		}
		words = w
	}
	return strings.Join(words, sep)
}

// CamelCaseToLower transform a camel-cased string into a
// lowercase string, separating the words with sep. e.g.
// FooBar with '_' as sep becomes foo_bar.
func CamelCaseToLower(s string, sep string) string {
	return CamelCaseToString(s, sep, strings.ToLower)
}

// CamelCaseToWords transforms a camel-cased string into a
// string, separating the words with sep.
func CamelCaseToWords(s string, sep string) string {
	return CamelCaseToString(s, sep, nil)
}

// Lines returns the lines in text between begin and begin+count,
// including both. Invalid line numbers (< 0 or > number of lines) are
// just ignored, so the number of returned lines might be different
// than count. If nonEmpty is true, empty lines will be removed from
// the output before selecting the specified lines. The string returned
// will have its lines separated by just the '\n' character. Any '\r'
// characters in the provided text will be removed.
func Lines(text string, begin int, count int, nonEmpty bool) string {
	s := strings.Replace(text, "\r", "", -1)
	lines := strings.Split(s, "\n")
	if nonEmpty {
		for ii := 0; ii < len(lines); ii++ {
			if lines[ii] == "" {
				lines = append(lines[:ii], lines[ii+1:]...)
				ii--
			}
		}
	}
	if begin < 0 {
		count += begin
		begin = 0
	}
	end := begin + count
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[begin:end], "\n")
}

// FileLines works like lines, but it reads the initial text from the
// given filename.
func FileLines(filename string, begin int, count int, nonEmpty bool) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return Lines(string(b), begin, count, nonEmpty), nil
}
