package facebook

import (
	"net/url"

	"gnd.la/net/httpclient"
	"gnd.la/net/oauth2"
)

const (
	Authorization = "https://www.facebook.com/dialog/oauth"
	Exchange      = "https://graph.facebook.com/oauth/access_token"
)

type App struct {
	*oauth2.Client
}

func (a *App) Parse(s string) error {
	return a.client().Parse(s)
}

func (a *App) client() *oauth2.Client {
	if a.Client == nil {
		a.Client = oauth2.New(Authorization, Exchange)
		a.Client.ResponseHasError = responseHasError
		a.Client.DecodeError = decodeResponseError
	}
	return a.Client
}

func (app *App) do(f func(string, url.Values, string) (*httpclient.Response, error), path string, form url.Values, accessToken string, out interface{}) error {
	endpoint := graphURL(path, accessToken)
	resp, err := f(endpoint, form, accessToken)
	if err != nil {
		return err
	}
	defer resp.Close()
	return resp.UnmarshalJSON(out)
}

func (a *App) Clone(ctx httpclient.Context) *App {
	ac := *a
	ac.Client = ac.Client.Clone(ctx)
	return &ac
}

func (app *App) Get(path string, data url.Values, accessToken string, out interface{}) error {
	return app.do(app.Client.Get, path, data, accessToken, out)
}

func (app *App) Post(path string, data url.Values, accessToken string, out interface{}) error {
	return app.do(app.Client.Post, path, data, accessToken, out)
}
