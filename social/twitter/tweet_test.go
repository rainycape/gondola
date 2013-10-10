package twitter

import (
	"gnd.la/config"
	"strings"
	"testing"
)

var (
	tests = map[string]string{
		"this doesn't need truncation":                                               "this doesn't need truncation",
		strings.Repeat("word ", 40) + "http://www.google.com":                        strings.Repeat(" word", 22)[1:] + " wor" + ellipsis + " http://www.google.com",
		strings.Repeat("a", 141):                                                     strings.Repeat("a", 138) + ellipsis,
		"this text will disappear " + strings.Repeat("https://www.google.com ", 100): strings.Repeat(" https://www.google.com", 5)[1:],
	}
)

type testConfig struct {
	TwitterApp   *App
	TwitterToken *Token
}

func loadCredentials(t *testing.T) (*App, *Token) {
	file := "credentials.conf"
	cfg := &testConfig{}
	if err := config.ParseFile(file, cfg); err != nil {
		t.Skipf("error parsing credentials: %s", err)
	}
	return cfg.TwitterApp, cfg.TwitterToken
}

func TestTruncate(t *testing.T) {
	for k, v := range tests {
		tr := truncateText(k, countCharacters(k), nil)
		t.Logf("truncated from %d characters to %d", countCharacters(k), countCharacters(tr))
		if tr != v {
			t.Errorf("error truncating %q. wanted %q, got %q", k, v, tr)
		}
	}
}

func TestTweet(t *testing.T) {
	app, token := loadCredentials(t)
	t.Log("Using app ", app, " and token ", token)
	opts := &TweetOptions{
		Truncate: true,
	}
	for k := range tests {
		tw, err := Update(k, app, token, opts)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Logf("Send tweet %s", tw.Id)
	}
}
