package messages

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// extractFormula takes a Plural Form expression e.g. "nplurals=2; plural=n == 1 ? 0 : 1;"
// and returns its formula (e.g. "n== 1 ? 0 : 1") as well as the number of plural
// forms (in the given example, 2). If the plural form can't be parsed, an error
// is returned.
func extractFormula(text string) (formula string, nplurals int, err error) {
	form := strings.TrimSpace(strings.ToLower(strings.Replace(text, "\\\n", "", -1)))
	if !strings.HasPrefix(form, "nplurals=") {
		err = fmt.Errorf("invalid Plural-Forms %q, not starting with nplurals=", text)
		return
	}
	form = form[9:]
	sep := strings.Index(form, ";")
	if sep == -1 {
		err = fmt.Errorf("invalid Plural-Forms %q, can't find number of plurals", text)
		return
	}
	nplurals, err = strconv.Atoi(form[:sep])
	if err != nil {
		err = fmt.Errorf("invalid Plural-Forms %q, error parsing nplurals: %s", text, err)
		return
	}
	form = strings.TrimSpace(form[sep+1:])
	if !strings.HasPrefix(form, "plural=") {
		err = fmt.Errorf("invalid plural formula %q, not starting with plural=", form)
		return
	}
	if form[len(form)-1] == ';' {
		form = form[:len(form)-1]
	}
	form = strings.TrimSpace(form[7:])
	if len(form) > 1 && form[0] == '(' && form[len(form)-1] == ')' {
		form = form[1 : len(form)-1]
	}
	formula = strings.TrimSpace(form)
	return
}

func funcFromFormula(form string) (string, error) {
	f, _, err := extractFormula(form)
	if err != nil {
		return "", err
	}
	var s scanner.Scanner
	s.Init(strings.NewReader(f))
	s.Error = func(s *scanner.Scanner, msg string) {
		err = fmt.Errorf("error parsing plural formula %s: %s", s.Pos(), msg)
	}
	s.Mode = scanner.ScanIdents | scanner.ScanInts
	s.Whitespace = 0
	tok := s.Scan()
	var code []string
	var buf bytes.Buffer
	for tok != scanner.EOF && err == nil {
		switch tok {
		case scanner.Ident, scanner.Int:
			buf.WriteString(s.TokenText())
		case '?':
			code = append(code, fmt.Sprintf("if %s {\n", buf.String()))
			buf.Reset()
		case ':':
			code = append(code, fmt.Sprintf("return %s\n}\n", buf.String()))
			buf.Reset()
		default:
			buf.WriteRune(tok)
		}
		tok = s.Scan()
	}
	if err != nil {
		return "", err
	}
	if len(code) == 0 && buf.Len() > 0 && buf.String() != "0" {
		code = append(code, fmt.Sprintf("if %s {\nreturn 1\n}\nreturn 0\n", buf.String()))
		buf.Reset()
	}
	if buf.Len() > 0 {
		code = append(code, fmt.Sprintf("\nreturn %s\n", buf.String()))
	}
	return strings.Join(code, ""), nil
}
