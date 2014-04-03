package stringutil

import (
	"bytes"
	"io"
	"os"
	"strings"
)

// ReadTextFile returns the contents of the file
// at the given path as a string.
func ReadTextFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ReadLines returns the non-empty lines from
// the file at the given path. The OS specific
// line separator is used to split the text into
// lines.
func ReadLines(path string) ([]string, error) {
	s, err := ReadTextFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(s, LineSeparator)
	nonEmpty := make([]string, 0, len(lines))
	for _, v := range lines {
		if v != "" {
			nonEmpty = append(nonEmpty, v)
		}
	}
	return nonEmpty, nil
}

// MustReadLines works like ReadLines, but panics if
// there's an error
func MustReadLines(path string) []string {
	lines, err := ReadLines(path)
	if err != nil {
		panic(err)
	}
	return lines
}
