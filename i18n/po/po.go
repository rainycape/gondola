package po

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
)

const (
	whitespace = 1<<'\t' | 1<<'\r' | 1<<' '
)

type Translation struct {
	Context      string
	Singular     string
	Plural       string
	Translations []string
}

type Po struct {
	Attrs    map[string]string
	Messages []*Translation
}

func (p *Po) addTranslation(t *Translation) {
	p.Messages = append(p.Messages, t)
}

func unexpected(s *scanner.Scanner, tok rune) error {
	return fmt.Errorf("unexpected %s token at %s: %s", scanner.TokenString(tok), s.Pos(), s.TokenText())
}

func readString(s *scanner.Scanner, tok *rune, err *error) string {
	val := ""
	*tok = s.Scan()
	var str string
	for *tok == scanner.String {
		str, *err = strconv.Unquote(s.TokenText())
		if *err != nil {
			break
		}
		val += str
		*tok = s.Scan()
	}
	return val
}

type namer interface {
	Name() string
}

func parsePo(r io.Reader, filename string) (*Po, error) {
	comment := false
	s := new(scanner.Scanner)
	var err error
	s.Init(r)
	s.Filename = filename
	s.Error = func(s *scanner.Scanner, msg string) {
		if !comment {
			err = fmt.Errorf("error parsing %s: %s", s.Pos(), msg)
		}
	}
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts
	tok := s.Scan()
	po := &Po{Attrs: make(map[string]string)}
	var trans *Translation
	for tok != scanner.EOF && err == nil {
		if tok == '#' {
			// Skip until EOL
			comment = true
			s.Whitespace = whitespace
			for tok != '\n' && tok != scanner.EOF {
				tok = s.Scan()
			}
			s.Whitespace = scanner.GoWhitespace
			comment = false
			tok = s.Scan()
			continue
		}
		if tok != scanner.Ident {
			err = unexpected(s, tok)
			break
		}
		text := s.TokenText()
		switch text {
		case "msgctxt":
			if trans != nil {
				if len(trans.Translations) == 0 {
					err = unexpected(s, tok)
					break
				}
				po.addTranslation(trans)
			}
			trans = &Translation{Context: readString(s, &tok, &err)}
		case "msgid":
			if trans != nil {
				if len(trans.Translations) > 0 || trans.Singular != "" {
					po.addTranslation(trans)
				} else if trans.Context != "" {
					trans.Singular = readString(s, &tok, &err)
					break
				}
			}
			trans = &Translation{Singular: readString(s, &tok, &err)}
		case "msgid_plural":
			if trans == nil || trans.Plural != "" {
				err = unexpected(s, tok)
				break
			}
			trans.Plural = readString(s, &tok, &err)
		case "msgstr":
			str := readString(s, &tok, &err)
			if tok == '[' {
				tok = s.Scan()
				if tok != scanner.Int {
					err = unexpected(s, tok)
					break
				}
				ii, _ := strconv.Atoi(s.TokenText())
				if ii != len(trans.Translations) {
					err = unexpected(s, tok)
					break
				}
				if tok = s.Scan(); tok != ']' {
					err = unexpected(s, tok)
					break
				}
				str = readString(s, &tok, &err)
			}
			trans.Translations = append(trans.Translations, str)
		default:
			err = unexpected(s, tok)
		}
	}
	if trans != nil {
		po.addTranslation(trans)
	}
	if err != nil {
		return nil, err
	}
	for _, v := range po.Messages {
		if v.Context == "" && v.Singular == "" {
			if len(v.Translations) > 0 {
				meta := v.Translations[0]
				for _, line := range strings.Split(meta, "\n") {
					colon := strings.Index(line, ":")
					if colon > 0 {
						key := strings.TrimSpace(line[:colon])
						value := strings.TrimSpace(line[colon+1:])
						po.Attrs[key] = value
					}
				}
			}
			break
		}
	}
	return po, nil
}

func Parse(r io.Reader) (*Po, error) {
	filename := ""
	if n, ok := r.(namer); ok {
		filename = n.Name()
	}
	return parsePo(r, filename)
}

func ParseFile(filename string) (*Po, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parsePo(f, filename)
}
