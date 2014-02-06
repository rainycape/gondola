package textutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

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
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading ini input: %s", err)
	}
	s := strings.Replace(strings.Replace(strings.Replace(string(data), "\\\n", "", -1), "\\\r\n", "", -1), "\r\n", "\n", -1)
	values := make(map[string]string)
	for ii, line := range strings.Split(s, "\n") {
		if line == "" || line[0] == ';' {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid line %d %q - missing = character", ii+1, line)
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			values[key] = value
		}

	}
	return values, nil
}
