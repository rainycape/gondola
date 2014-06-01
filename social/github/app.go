package github

import (
	"net/url"

	"gnd.la/net/httpclient"
	"gnd.la/net/oauth2"
	"gnd.la/net/urlutil"
)

const (
	Authorization = "https://github.com/login/oauth/authorize"
	Exchange      = "https://github.com/login/oauth/access_token"
	Base          = "https://api.github.com"
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
		a.Client = oauth2.New(Authorization, Exchange)
		a.Client.DecodeError = decodeError
	}
	return a.Client
}

func (a *App) do(f func(string, url.Values, string) (*httpclient.Response, error), path string, form url.Values, accessToken string, out interface{}) error {
	u := a.URL(path)
	resp, err := f(u, form, accessToken)
	if err != nil {
		return err
	}
	defer resp.Close()
	return resp.UnmarshalJSON(out)
}

func (a *App) URL(path string) string {
	return urlutil.MustJoin(Base, path)
}

func (a *App) Get(path string, data url.Values, accessToken string, out interface{}) error {
	return a.do(a.Client.Get, path, data, accessToken, out)
}

func (a *App) Post(path string, data url.Values, accessToken string, out interface{}) error {
	return a.do(a.Client.Post, path, data, accessToken, out)
}
