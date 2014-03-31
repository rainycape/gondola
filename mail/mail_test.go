package mail

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
	"text/template"
)

var (
	boundaryRe = regexp.MustCompile("Gondola\\-Boundary\\-\\w+")
	tmpl       = template.Must(template.New("tmpl").Parse("{{ .foo }}"))
)

func testCredentials(t *testing.T, addr, server, username, password string, cram bool) {
	cr, user, passwd, host := parseServer(addr)
	if cr != cram || server != host || user != username || password != passwd {
		t.Errorf("Expecting %v, %v, %v, %v, got %v, %v, %v, %v",
			server, username, password, cram, host, user, passwd, cr)
	}
}

func TestCredentials(t *testing.T) {
	testCredentials(t, "smtp.example.com", "smtp.example.com", "", "", false)
	testCredentials(t, "pepe:lotas@smtp.example.com", "smtp.example.com", "pepe", "lotas", false)
	testCredentials(t, "cram?pepe:lotas@smtp.example.com", "smtp.example.com", "pepe", "lotas", true)
	testCredentials(t, "invalid?pepe:lotas@smtp.example.com", "smtp.example.com", "invalid?pepe", "lotas", false)
	testCredentials(t, "pepe@lotas.com:mayonesa@smtp.example.com", "smtp.example.com", "pepe@lotas.com", "mayonesa", false)
}

type Validation struct {
	Address    string
	Email      string
	UseNetwork bool
	Valid      bool
}

func TestValidation(t *testing.T) {
	cases := []Validation{
		{"pepe  @gmail.com", "", true, false},
		{"pepe@lotas@gmail.com", "", true, false},
		{"pepe", "", true, false},
		{"pepe@", "", true, false},
		{"@gmail.com", "", true, false},
		{"pepe@gmail.com", "", true, true},
		{"Pepe <pepe@gmail.com>", "pepe@gmail.com", true, true},
		{"fiam@abra.rm-fr.net", "", true, true},
		{"pepe@gmaildoesnotexistwolololhopefullynooneregistersthisdomainandbreaksthistest.com", "", false, true},
		{"pepe@gmaildoesnotexistwolololhopefullynooneregistersthisdomainandbreaksthistest.com", "", true, false},
	}
	for _, v := range cases {
		email, err := Validate(v.Address, v.UseNetwork)
		t.Logf("Validated address %q (net %v), error: %v", v.Address, v.UseNetwork, err)
		valid := err == nil
		if valid != v.Valid {
			e := "valid"
			if valid {
				e = "invalid"
			}
			t.Errorf("Error validating %q (net %v), expecting %s address", v.Address, v.UseNetwork, e)
		}
		if v.Email != "" && v.Email != email {
			t.Errorf("invalid email %q from %q, want %q", email, v.Address, v.Email)
		}
	}
}

type EmailTest struct {
	To      []string
	Options *Options
	Expect  string
}

func makeAttachments(b []byte) []*Attachment {
	att, err := NewAttachment("", "", bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return []*Attachment{att}

}

var (
	emailTests = []EmailTest{
		{
			Options: &Options{
				Message: "foo",
			},
			Expect: "To: go@example.com\n\nFrom: go@example.com\r\n\r\nfoo\n",
		},
		{
			Options: &Options{
				Message: []byte("foo"),
			},
			Expect: "To: go@example.com\n\nFrom: go@example.com\r\n\r\nfoo\n",
		},
		{
			Options: &Options{
				Message: bytes.NewReader([]byte("foo")),
			},
			Expect: "To: go@example.com\n\nFrom: go@example.com\r\n\r\nfoo\n",
		},
		{
			Options: &Options{
				Message: tmpl,
				Data:    map[string]string{"foo": "bar"},
			},
			Expect: "To: go@example.com\n\nFrom: go@example.com\r\n\r\nbar\n",
		},
		{
			Options: &Options{
				Message:     "foo",
				Attachments: makeAttachments([]byte("bar")),
			},
			Expect: "To: go@example.com\n\nFrom: go@example.com\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=Gondola-Boundary-A\r\n--Gondola-Boundary-A\nContent-Type: text/plain; charset=utf-8\r\nfoo\r\n\r\n--Gondola-Boundary-A\r\nContent-Type: application/octet-stream\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"file\"\r\n\r\nYmFy\r\n--Gondola-Boundary-A--\n",
		},
	}
)

func replaceBoundary(s string) string {
	return boundaryRe.ReplaceAllString(s, "Gondola-Boundary-A")
}

func TestSendEmail(t *testing.T) {
	p := printer
	defer func() {
		printer = p
	}()
	var res string
	printer = func(format string, args ...interface{}) (int, error) {
		res = fmt.Sprintf(format, args...)
		return len(res), nil
	}
	for _, v := range emailTests {
		if v.To == nil {
			v.To = []string{"go@example.com"}
		}
		if v.Options.From == "" {
			v.Options.From = "go@example.com"
		}
		v.Options.Server = "echo"
		if err := Send(v.To, v.Options); err != nil {
			t.Error(err)
			continue
		}
		// boundary is random, so we need to change it
		res = replaceBoundary(res)
		if res != v.Expect {
			t.Errorf("options %v expecting email %q, got %q instead", v, v.Expect, res)
		}
	}
}
