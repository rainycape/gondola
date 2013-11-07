package html

import (
	"bufio"
	"bytes"
	"io"
)

const (
	stateSkip = iota
	stateAppend
	stateWhitespace
)

var (
	textarea    = []byte("textarea>")
	pre         = []byte("pre>")
	textareaEnd = []byte("/textarea>")
	preEnd      = []byte("/pre>")
)

func beginVerbatim(r *bufio.Reader) bool {
	b, _ := r.Peek(len(textarea))
	if bytes.Equal(b, textarea) {
		return true
	}
	if b == nil {
		var err error
		b, err = r.Peek(len(pre))
		if err != nil {
			return false
		}
	}
	b = b[:len(pre)]
	return bytes.Equal(b, pre)
}

func endVerbatim(r *bufio.Reader) bool {
	b, _ := r.Peek(len(textareaEnd))
	if bytes.Equal(b, textareaEnd) {
		return true
	}
	if b == nil {
		var err error
		b, err = r.Peek(len(preEnd))
		if err != nil {
			return false
		}
	}
	b = b[:len(preEnd)]
	return bytes.Equal(b, preEnd)
}

// Minify removes insignificant whitespace from the
// given HTML. Please, keep in mind that this function could
// break your HTML if you're using embedded scripts or if you
// rely on automatical semicolon insertion, because multiple
// whitespaces (' ', '\n', '\t' and '\r') will be collapsed into
// a single ' ' character. Formatting inside pre and textarea tags
// is preserved.
func Minify(w io.Writer, r io.Reader) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	state := stateSkip
	verbatim := 0
	skipped := false
	for {
		c, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch c {
		case ' ', '\n', '\t', '\r':
			skipped = true
			if state == stateAppend {
				skipped = false
				if verbatim > 0 {
					bw.WriteByte(c)
				} else {
					bw.WriteByte(' ')
					state = stateWhitespace
				}
			}
		case '<':
			bw.WriteByte('<')
			if beginVerbatim(br) {
				verbatim++
			} else if endVerbatim(br) {
				verbatim--
			}
			state = stateAppend
			skipped = false
		case '>':
			bw.WriteByte('>')
			if verbatim == 0 {
				state = stateSkip
			}
			skipped = false
		default:
			if skipped && state != stateWhitespace {
				bw.WriteByte(' ')
			}
			bw.WriteByte(c)
			state = stateAppend
			skipped = false
		}
	}
	return bw.Flush()
}
