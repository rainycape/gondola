package pinterest

import (
	"io/ioutil"
	"strings"
	"testing"
)

func parseAccount(tb testing.TB) *Account {
	acc := &Account{}
	if data, err := ioutil.ReadFile("account.txt"); err == nil {
		if err := acc.Parse(strings.TrimSpace(string(data))); err == nil {
			tb.Logf("Using account %+v", *acc)
			return acc
		}
	}
	tb.Skip("Please, place your Pinterest account in account.txt (username: password)")
	return nil
}

func TestSignIn(t *testing.T) {
	acc := parseAccount(t)
	sess, err := acc.SignIn()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Session %+v", sess)
	boards, err := sess.Boards()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range boards {
		t.Logf("Board %+v", *v)
	}
}

func TestPost(t *testing.T) {
	acc := parseAccount(t)
	sess, err := acc.SignIn()
	if err != nil {
		t.Fatal(err)
	}
	boards, err := sess.Boards()
	if err != nil {
		t.Fatal(err)
	}
	pin, err := sess.Post(boards[0], &Pin{
		Link:        "http://cuteanimals.me",
		Image:       "http://cuteanimals.me/-img/52770a211605fb1528000003.jpg",
		Description: "Just testing",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Pin %+v", pin)
}
