package textutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// IniOptions specify the options for ParseIniOptions.
type IniOptions struct {
	// Separator indicates the string used as key-value separator.
	// If empty, "=" is used.
	Separator string
	// Comment indicates the string used to check if a line is a comment.
	// Lines starting with this string are ignored. If empty, all lines
	// are parsed.
	Comment string
}

// ParseIni parses a .ini style file in the form:
//
//  key1 = value
//  key2 = value
//
// Values that contain newlines ("\n" or "\r\n") need to
// escape them by ending the previous line with a '\'
// character. Lines starting with the ';' character are
// considered comments and ignored. Empty lines are ignored
// too. If a non-empty, non-comment line does not contain
// a '=' an error is returned.
func ParseIni(r io.Reader) (map[string]string, error) {
	return ParseIniOptions(r, nil)
}

// ParseIniOptions works like ParseIni, but allows the caller to specify
// the strings which represent separators and comments. If opts is nil, this
// function acts like ParseIni. If Separator is empty, it defaults to '='. If
// Comment is empty, no lines are considered comments.
func ParseIniOptions(r io.Reader, opts *IniOptions) (map[string]string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading ini input: %s", err)
	}
	s := strings.Replace(strings.Replace(strings.Replace(string(data), "\\\n", "", -1), "\\\r\n", "", -1), "\r\n", "\n", -1)
	var separator string
	var comment string
	if opts != nil {
		separator = opts.Separator
		comment = opts.Comment
	} else {
		comment = ";"
	}
	if separator == "" {
		separator = "="
	}
	values := make(map[string]string)
	for ii, line := range strings.Split(s, "\n") {
		if line == "" || (comment != "" && strings.HasPrefix(line, comment)) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			parts := strings.SplitN(trimmed, separator, 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid line %d %q - missing separator %q", ii+1, line, separator)
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			values[key] = value
		}

	}
	return values, nil
}
