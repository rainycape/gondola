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
	sess, err := SignIn(acc)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Session %+v", sess)
}

func TestPost(t *testing.T) {
	acc := parseAccount(t)
	sess, err := SignIn(acc)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Session %+v", sess)
	boards, err := Boards(sess)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range boards {
		t.Logf("Board %+v", *v)
	}
	pin, err := Post(sess, boards[0], &Pin{
		Link:        "http://cuteanimals.me",
		Image:       "http://cuteanimals.me/-img/52770a211605fb1528000003.jpg",
		Description: "Just testing",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Pin %+v", pin)
}
