// Package runtimeutil contains some utility functions for
// formatting stack traces and source code.
package runtimeutil

import (
	"fmt"
	"gnd.la/html"
	"gnd.la/util/stringutil"
	"html/template"
	"runtime"
	"strconv"
	"strings"
)

// FormatStack returns the current call stack formatted
// as a string. The skip argument indicates how many frames
// should be omitted.
func FormatStack(skip int) string {
	return formatStack(skip, false)
}

// FormatStackHTML works like FormatStack, but returns its
// results in HTML, using <abbr> tags for displaying
// pointer contents (when possible).
func FormatStackHTML(skip int) template.HTML {
	s := formatStack(skip, true)
	return template.HTML(s)
}

func formatStack(skip int, _html bool) string {
	// Always skip the frames for formatStack and FormatStack(HTML)
	skip += 2
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
	lines = prettyStack(lines, _html)
	return strings.Join(lines, "\n")
}

// FormatCaller finds the caller, skipping skip frames, and then formats
// the source using FormatSource. The location is returned in the first
// string, while the formatted source is the second return parameter.
// See the documentation for FormatSource() for an explanation of the
// rest of the parameters.
func FormatCaller(skip int, count int, numbers bool, highlight bool) (string, string) {
	return formatCaller(skip, count, numbers, highlight, false)
}

// FormatCallerHTML works like FormatCaller, but uses FormatSourceHTML
// rather than FormatSource
func FormatCallerHTML(skip int, count int, numbers bool, highlight bool) (string, template.HTML) {
	location, source := formatCaller(skip, count, numbers, highlight, true)
	return location, template.HTML(source)
}

func formatCaller(skip int, count int, numbers bool, highlight bool, _html bool) (string, string) {
	// Always skip the frames for formatCaller and FormatCaller(HTML)
	skip += 2
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", ""
	}
	location := fmt.Sprintf("%s, line %d", file, line)
	source, _ := formatSource(file, line, count, numbers, highlight, _html)
	return location, source
}

// FormatSource returns the source from filename around line formatted as
// HTML. The count parameter indicates the number of lines to include before
// and after the target line (if possible). If numbers is true, the line numbers
// will be preprended to each line. Finally, if highlight is true, the line
// passed as the second parameter with be highlighted by adding the string
// "<===" at its end.
func FormatSource(filename string, line int, count int, numbers bool, highlight bool) (string, error) {
	return formatSource(filename, line, count, numbers, highlight, false)
}

// FormatSourceHTML works like FormatSource, but returns the result as HTML. Highlighting
// is done by wrapping the line inside a span element with its class set to "current".
func FormatSourceHTML(filename string, line int, count int, numbers bool, highlight bool) (template.HTML, error) {
	s, err := formatSource(filename, line, count, numbers, highlight, true)
	return template.HTML(s), err
}

func formatSource(filename string, line int, count int, numbers bool, highlight bool, _html bool) (string, error) {
	begin := line - count - 1
	count = count*2 + 1
	if begin < 0 {
		count += begin
		begin = 0
	}
	source, err := stringutil.FileLines(filename, begin, count, false)
	if err != nil {
		return "", err
	}

	var format string
	if numbers {
		// Line numbers start at 1
		begin++
		maxLen := len(strconv.Itoa(begin + count))
		format = fmt.Sprintf("%%%dd: %%s", maxLen)
	}
	slines := strings.Split(source, "\n")
	for ii, v := range slines {
		if numbers {
			v = fmt.Sprintf(format, begin, v)
		}
		if _html {
			v = html.Escape(v)
		}
		if highlight && begin == line {
			if _html {
				v = fmt.Sprintf("<span class=\"current\">%s</span>", v)
			} else {
				v += " <==="
			}
		}
		slines[ii] = v
		begin++
	}
	return strings.Join(slines, "\n"), nil
}

// GetPanic returns the number of frames to skip and the PC
// for the uppermost panic in the call stack (there might be
// multiple panics when a recover() catches a panic and then
// panics again). The second value indicates how many stack frames
// should be skipped in the stacktrace (they might not always match).
// The last return value indicates a frame could be found.
func GetPanic() (int, int, uintptr, bool) {
	skip := 0
	callers := make([]uintptr, 10)
	for {
		calls := callers[:runtime.Callers(skip, callers)]
		c := len(calls)
		if c == 0 {
			break
		}
		for ii := c - 1; ii >= 0; ii-- {
			f := runtime.FuncForPC(calls[ii])
			if f != nil {
				name := f.Name()
				if strings.HasPrefix(name, "runtime.") && strings.Contains(name, "panic") {
					pcSkip := skip + ii - 1
					stackSkip := pcSkip
					switch name {
					case "runtime.panic":
					case "runtime.sigpanic":
						stackSkip -= 2
					default:
						stackSkip--
					}
					return pcSkip, stackSkip, calls[ii], true
				}
			}
		}
		skip += c
	}
	return 0, 0, 0, false
}
