package google

import (
	"errors"
	"net/url"

	"gnd.la/net/httpclient"
	"gnd.la/net/oauth2"
)

const (
	PlusScope    = "https://www.googleapis.com/auth/plus.login"
	EmailScope   = "https://www.googleapis.com/auth/plus.profile.emails.read"
	ProfileScope = "profile"

	authorizationURL = "https://accounts.google.com/o/oauth2/auth"
	tokenURL         = "https://accounts.google.com/o/oauth2/token"
)

var (
	errEmptyState = errors.New("google authentication state can't be empty")
)

type App struct {
	*oauth2.Client
}

func (a *App) Parse(s string) error {
	return a.client().Parse(s)
}

func (a *App) Clone(ctx httpclient.Context) *App {
	ac := *a
	ac.Client = ac.Client.Clone(ctx)
	return &ac
}

func (a *App) client() *oauth2.Client {
	if a.Client == nil {
		a.Client = oauth2.New(authorizationURL, tokenURL)
		a.Client.DecodeError = googleError
		a.Client.AuthorizationParameters = map[string]string{
			"response_type":          "code",
			"include_granted_scopes": "true",
			"access_type":            "offline",
		}
		a.Client.ExchangeParameters = map[string]string{
			"grant_type": "authorization_code",
		}
		a.Client.ScopeSeparator = " "
	}
	return a.Client
}

func (a *App) Refresh(refresh string) (*oauth2.Token, error) {
	values := make(url.Values)
	values.Add("refresh_token", refresh)
	values.Add("client_id", a.Id)
	values.Add("client_secret", a.Secret)
	values.Add("grant_type", "refresh_token")
	resp, err := a.client().HTTPClient.PostForm(tokenURL, values)
	if err != nil {
		return nil, err
	}
	tok, err := oauth2.NewToken(resp)
	if err != nil {
		return nil, err
	}
	tok.Refresh = refresh
	return tok, nil
}
