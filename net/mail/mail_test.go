package mail

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	Message *Message
	Expect  string
}

func makeAttachments(file string, contentId string) []*Attachment {
	f, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		panic(err)
	}
	att, err := NewAttachment(file, f)
	if err != nil {
		panic(err)
	}
	att.ContentID = contentId
	return []*Attachment{att}
}

var (
	sendTests = []*Message{
		&Message{
			TextBody: "foo",
		},
		&Message{
			TextBody:    "foo",
			Attachments: makeAttachments("lenna.jpg", ""),
		},
		&Message{
			TextBody:    "This is lenna",
			HTMLBody:    "<b>THIS IS LENNA</b>",
			Attachments: makeAttachments("lenna.jpg", ""),
		},
		&Message{
			TextBody:    "This is lenna",
			HTMLBody:    "<html><body>LENNA <br><img src=\"cid:LENNA\" alt=\"This is Lenna\"><br><b>EMBEDDED</b></body></html>",
			Attachments: makeAttachments("lenna.jpg", "LENNA"),
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
	count := 0
	printer = func(format string, args ...interface{}) (int, error) {
		res = fmt.Sprintf(format, args...)
		// This is useful when adding new tests
		ioutil.WriteFile(filepath.Join("testdata", fmt.Sprintf("out.%d.eml", count)), []byte(res), 0644)
		count++
		return len(res), nil
	}
	for ii, v := range sendTests {
		if v.To == nil {
			v.To = []string{"receiver@example.com"}
		}
		if v.From == "" {
			v.From = "sender@example.com"
		}
		v.Server = "echo"
		if err := Send(v); err != nil {
			t.Error(err)
			continue
		}
		// boundary is random, so we need to change it
		res = replaceBoundary(res)
		path := filepath.Join("testdata", fmt.Sprintf("expect.%d.eml", ii))
		data, err := ioutil.ReadFile(path)
		if err != nil {
			t.Error(err)
			continue
		}
		if res != string(data) {
			t.Errorf("message %v expecting email %q, got %q instead", v, string(data), res)
		}
	}
}
