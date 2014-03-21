package google

import (
	"encoding/json"
	"errors"
	"gnd.la/util/textutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	PlusScope    = "https://www.googleapis.com/auth/plus.login"
	EmailScope   = "https://www.googleapis.com/auth/plus.profile.emails.read"
	ProfileScope = "profile"

	authorizationURL = "https://accounts.google.com/o/oauth2/auth?"
	tokenURL         = "https://accounts.google.com/o/oauth2/token"
)

var (
	errEmptyState = errors.New("google authentication state can't be empty")
)

type App struct {
	Key    string
	Secret string
}

type Token struct {
	Key     string
	Expires time.Time
	Refresh string
}

func (a *App) Parse(s string) error {
	fields, err := textutil.SplitFieldsOptions(s, ":", &textutil.SplitOptions{ExactCount: 2})
	if err != nil {
		return err
	}
	a.Key = fields[0]
	a.Secret = fields[1]
	return nil
}

func (a *App) Authorize(scope []string, redirect string, state string) (string, error) {
	if state == "" {
		return "", errEmptyState
	}
	if len(scope) == 0 {
		scope = []string{PlusScope}
	}
	values := make(url.Values)
	values.Add("client_id", a.Key)
	values.Add("redirect_uri", redirect)
	values.Add("scope", strings.Join(scope, " "))
	values.Add("state", state)
	values.Add("response_type", "code")
	values.Add("cookie_policy", "single_host_origin")
	values.Add("include_granted_scopes", "true")
	values.Add("access_type", "offline")

	return authorizationURL + values.Encode(), nil
}

func (a *App) Exchange(code string, redirect string) (*Token, error) {
	values := make(url.Values)
	values.Add("code", code)
	values.Add("client_id", a.Key)
	values.Add("client_secret", a.Secret)
	values.Add("redirect_uri", redirect)
	values.Add("grant_type", "authorization_code")
	resp, err := http.PostForm(tokenURL, values)
	if err != nil {
		return nil, err
	}
	return decodeToken(resp, "")
}

func (a *App) Refresh(refresh string) (*Token, error) {
	values := make(url.Values)
	values.Add("refresh_token", refresh)
	values.Add("client_id", a.Key)
	values.Add("client_secret", a.Secret)
	values.Add("grant_type", "refresh_token")
	resp, err := http.PostForm(tokenURL, values)
	if err != nil {
		return nil, err
	}
	return decodeToken(resp, refresh)
}

func decodeToken(resp *http.Response, refresh string) (*Token, error) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, googleError(resp.Body, resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if refresh == "" {
		refresh, _ = m["refresh_token"].(string)
	}
	expires := m["expires_in"].(float64)
	return &Token{
		Key:     m["access_token"].(string),
		Refresh: refresh,
		Expires: time.Now().Add(time.Duration(expires) * time.Second),
	}, nil
}
