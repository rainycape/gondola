package oauth

import (
	"io/ioutil"
	"net/url"
	"testing"
)

var echo = "http://term.ie/oauth/example/echo_api.php"

// These tests use the testing oAuth server
// documented at http://term.ie/oauth/example/

func testOAuth(t *testing.T, method string, values url.Values) {
	c := &Consumer{
		Key:             "key",
		Secret:          "secret",
		RequestTokenURL: "http://term.ie/oauth/example/request_token.php",
		AccessTokenURL:  "http://term.ie/oauth/example/access_token.php",
		CallbackURL:     "oob",
	}
	_, rt, err := c.Authorization()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("request token is %+v", rt)
	at, err := c.Exchange(rt, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("access token is %+v", at)
	t.Logf("sending request with values %+v", values)
	resp, err := c.SendRequest(method, echo, values, at)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	t.Logf("server replied with %q", s)
	if len(values) == 0 {
		if s != "" {
			t.Errorf("expected empty response, got %q", s)
		}
	} else {
		if e := values.Encode(); e != s {
			t.Errorf("expected %q got %q", e, s)
		}
	}
}

func TestOAuth(t *testing.T) {
	testOAuth(t, "GET", nil)
}

func TestGet(t *testing.T) {
	testOAuth(t, "GET", url.Values{"foo": []string{"bar"}})
}

func TestPost(t *testing.T) {
	testOAuth(t, "POST", url.Values{"foo": []string{"bar"}})
}

func TestUnicode(t *testing.T) {
	testOAuth(t, "POST", url.Values{"alberto": []string{"garcía"}})
}
