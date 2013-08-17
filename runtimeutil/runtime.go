// Package runtimeutil contains some utility functions for
// formatting stack traces and source code.
package runtimeutil

import (
	"fmt"
	"gondola/html"
	"gondola/util"
	"html/template"
	"runtime"
	"strconv"
	"strings"
)

// FormatStack returns the current call stack formatted
// as a string. The skip argument indicates how many frames
// should be omitted.
func FormatStack(skip int) string {
	const size = 8192
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	// Remove 2 * skip lines after first line, since they correspond
	// to the skipped frames
	lines := strings.Split(string(buf), "\n")
	end := 2*skip + 1
	if end > len(lines) {
		end = len(lines)
	}
	lines = append(lines[:1], lines[end:]...)
	lines = prettyStack(lines)
	return strings.Join(lines, "\n")
}

// FormatCaller finds the caller, skipping skip frames, and then formats
// the source using FormatSource. The count argument is passed to
// FormatSource without any modification.
func FormatCaller(skip int, count int) (location string, source template.HTML) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return
	}
	location = fmt.Sprintf("%s, line %d", file, line)
	source, _ = FormatSource(file, line, count)
	return
}

// FormatSource returns the source from filename around line formatted as
// HTML. The count parameter indicates the number of lines to include before
// and after the target line (if possible). The current line is wrapped
// inside a span element with its class set to "current".
func FormatSource(filename string, line int, count int) (template.HTML, error) {
	begin := line - count - 1
	count = count*2 + 1
	if begin < 0 {
		count += begin
		begin = 0
	}
	source, err := util.FileLines(filename, begin, count, false)
	if err != nil {
		return template.HTML(""), err
	}

	// Line numbers start at 1
	begin++
	maxLen := len(strconv.Itoa(begin + count))
	format := fmt.Sprintf("%%%dd: %%s", maxLen)
	slines := strings.Split(source, "\n")
	for ii, v := range slines {
		formatted := fmt.Sprintf(format, begin, html.Escape(v))
		if begin == line {
			formatted = fmt.Sprintf("<span class=\"current\">%s</span>", formatted)
		}
		slines[ii] = formatted
		begin++
	}
	return template.HTML(strings.Join(slines, "\n")), nil
}
