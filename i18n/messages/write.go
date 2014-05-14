package messages

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

const (
	maxLineLength = 80
)

var (
	newLine = []byte{'\n'}
)

func writeString(w io.Writer, prefix, str string) error {
	quoted := fmt.Sprintf("%q", str)
	if len(quoted)+len(prefix)+2 < maxLineLength {
		// No splitting
		_, err := io.WriteString(w, fmt.Sprintf("%s %s\n", prefix, quoted))
		return err
	}
	// Splitting
	if _, err := io.WriteString(w, fmt.Sprintf("%s \"\"\n", prefix)); err != nil {
		return err
	}
	quoted = quoted[1 : len(quoted)-1]
	return writeSuffixLines(w, "\"", "\"", quoted)
}

func startLine(w io.Writer, prefix, suffix string, nl bool) (int, error) {
	if nl {
		if _, err := io.WriteString(w, suffix); err != nil {
			return 0, err
		}
		if _, err := w.Write(newLine); err != nil {
			return 0, err
		}
	}
	return io.WriteString(w, prefix)
}

func writeLines(w io.Writer, prefix, str string) error {
	return writeSuffixLines(w, prefix, "", str)
}

func writeSuffixLines(w io.Writer, prefix, suffix, str string) error {
	count, err := startLine(w, prefix, suffix, false)
	if err != nil {
		return err
	}
	sl := len(suffix)
	bs := []byte(str)
	t := len(bs)
	nl := true
	ii := 0
	for ii < t {
		b := bs[ii]
		if nl {
			if b == ' ' || b == '\t' {
				ii++
				continue
			}
			nl = false
		}
		slice := bs[ii:]
		next := bytes.IndexAny(slice, " \n")
		if next == 0 {
			ii++
			continue
		}
		if next == -1 {
			next = len(slice) - 1
		}
		if count+sl+next >= maxLineLength {
			count, err = startLine(w, prefix, suffix, true)
			if err != nil {
				return err
			}
			nl = true
		}
		c, err := w.Write(slice[:next+1])
		if err != nil {
			return err
		}
		count += c
		ii += c
		if slice[next] == '\n' {
			count, err = startLine(w, prefix, suffix, false)
			if err != nil {
				return err
			}
			nl = true
		}
	}
	if suffix != "" {
		if _, err := io.WriteString(w, suffix); err != nil {
			return err
		}
	}
	_, err = w.Write(newLine)
	return err
}

func Write(w io.Writer, messages []*Message) error {
	for ii, m := range messages {
		if m.TranslatorComment != "" {
			if err := writeLines(w, "# ", m.TranslatorComment); err != nil {
				return err
			}
		}
		if len(m.Positions) > 1 {
			var comments []string
			positions := make([]string, len(m.Positions))
			for ii, v := range m.Positions {
				s := v.String()
				if v.Comment != "" {
					comments = append(comments, fmt.Sprintf("(%s) %s", s, v.Comment))
				}
				positions[ii] = s
			}
			if comments != nil {
				if err := writeLines(w, "#. ", strings.Join(comments, "\n")); err != nil {
					return err
				}
			}
			if err := writeLines(w, "#: ", strings.Join(positions, " ")); err != nil {
				return err
			}
		} else {
			p := m.Positions[0]
			if p.Comment != "" {
				if err := writeLines(w, "#. ", p.Comment); err != nil {
					return err
				}
			}
			if err := writeLines(w, "#: ", p.String()); err != nil {
				return err
			}
		}
		if m.Context != "" {
			if _, err := io.WriteString(w, fmt.Sprintf("msgctxt %q\n", m.Context)); err != nil {
				return err
			}
		}
		if err := writeString(w, "msgid", m.Singular); err != nil {
			return err
		}
		if m.Plural != "" {
			if err := writeString(w, "msgid_plural", m.Plural); err != nil {
				return err
			}
			tn := 2
			tl := len(m.Translations)
			if tl > tn {
				tn = tl
			}
			for ii := 0; ii < tn; ii++ {
				msgstr := ""
				if ii < tl {
					msgstr = m.Translations[ii]
				}
				if err := writeString(w, fmt.Sprintf("msgstr[%d]", ii), msgstr); err != nil {
					return err
				}
			}
		} else {
			msgstr := ""
			if len(m.Translations) > 0 {
				msgstr = m.Translations[0]
			}
			if err := writeString(w, "msgstr", msgstr); err != nil {
				return err
			}
		}
		if ii != len(messages)-1 {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return err
			}
		}
	}
	return nil
}
